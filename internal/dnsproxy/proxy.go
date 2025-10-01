package dnsproxy

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/firewall"
	"github.com/bavix/outway/internal/lanresolver"
	"github.com/bavix/outway/internal/localzone"
	"github.com/bavix/outway/internal/metrics"
	"github.com/bavix/outway/internal/version"
)

var (
	errProxyConfigurationNil      = errors.New("proxy configuration is nil")
	errNilDNSMessageForDoH        = errors.New("nil DNS message for DoH")
	errDoHClientNotInitialized    = errors.New("DoH client not initialized")
	errDoHStatus                  = errors.New("doh status")
	errInvalidUpstream            = errors.New("invalid upstream")
	errAtLeastOneUpstreamRequired = errors.New("at least one upstream is required")
)

const (
	defaultDNSTimeout  = 2 * time.Second
	defaultDoHTimeout  = 5 * time.Second
	defaultMinTTL      = 30
	protocolSplitLimit = 2
	addressSplitLimit  = 3

	// Protocol constants.
	protocolUDP = "udp"
	protocolTCP = "tcp"
	protocolDOH = "doh"
	protocolDOQ = "doq"
	protocolDOT = "dot"

	// Default fallback.
	defaultDNS = "8.8.8.8:53"
)

// legacy in-proxy cache removed in favor of cache decorator

// matchDomainPattern matches a hostname against a wildcard pattern like *.example.com
// Exact match if no wildcard.
// Enhanced for OpenWrt compatibility - handles various DNS query formats.
func matchDomainPattern(pattern, host string) bool {
	// Normalize both pattern and host for consistent comparison
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	host = strings.ToLower(strings.TrimSpace(host))

	// Remove trailing dots (DNS FQDN format)
	pattern = strings.TrimSuffix(pattern, ".")
	host = strings.TrimSuffix(host, ".")

	if pattern == "" || pattern == "*" {
		return true
	}

	if suffix, ok := strings.CutPrefix(pattern, "*."); ok {
		// Match both the exact domain (example.com) and subdomains (www.example.com)
		// This handles OpenWrt DNS queries properly
		return host == suffix || strings.HasSuffix(host, "."+suffix)
	}

	return pattern == host
}

type RuleStore struct {
	mu    sync.RWMutex
	rules []config.Rule
}

func NewRuleStore(rules []config.Rule) *RuleStore { return &RuleStore{rules: slices.Clone(rules)} }

func (s *RuleStore) List() []config.Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return slices.Clone(s.rules)
}

func (s *RuleStore) Upsert(r config.Rule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.rules {
		if s.rules[i].Pattern == r.Pattern {
			s.rules[i] = r

			return
		}
	}

	s.rules = append(s.rules, r)
}

func (s *RuleStore) Delete(pattern string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := s.rules[:0]
	for _, r := range s.rules {
		if r.Pattern != pattern {
			res = append(res, r)
		}
	}

	s.rules = res
}

func (s *RuleStore) FindIface(host string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.rules {
		if matchDomainPattern(r.Pattern, host) {
			return r.Via
		}
	}

	return ""
}

func (s *RuleStore) Find(host string) (config.Rule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.rules {
		if matchDomainPattern(r.Pattern, host) {
			return r, true
		}
	}

	return config.Rule{}, false
}

type Proxy struct {
	cfg       *config.Config
	backend   firewall.Backend
	upstreams []string
	rules     *RuleStore
	// removed legacy cache fields (using decorator now)
	active      atomic.Value // Resolver
	persistMu   sync.Mutex
	histMu      sync.Mutex
	histBuf     []QueryEvent
	histCap     int
	histHead    int
	histSize    int
	dnsUDP      *dns.Client
	dnsTCP      *dns.Client
	dohClient   *http.Client
	lanResolver *lanresolver.LANResolver
	lanWatcher  *localzone.Watcher
}

// ResolverActive returns the current active resolver atomically.
func (p *Proxy) ResolverActive() Resolver { //nolint:ireturn
	if v := p.active.Load(); v != nil {
		if r, ok := v.(Resolver); ok {
			return r
		}
	}

	return nil
}

func New(cfg *config.Config, backend firewall.Backend) *Proxy {
	capacity := cfg.History.MaxEntries
	if capacity <= 0 {
		capacity = 500
	}

	p := &Proxy{
		cfg:       cfg,
		backend:   backend,
		upstreams: cfg.GetUpstreamAddresses(),
		rules:     NewRuleStore(cfg.GetAllRules()),
		histCap:   capacity,
		dnsUDP:    &dns.Client{Net: "udp", Timeout: defaultDNSTimeout},
		dnsTCP:    &dns.Client{Net: "tcp", Timeout: defaultDNSTimeout},
		dohClient: &http.Client{Timeout: defaultDoHTimeout},
	}
	p.histBuf = make([]QueryEvent, p.histCap)
	// cache provided by decorator

	return p
}

func (p *Proxy) Start(ctx context.Context) error {
	if p.cfg == nil {
		return errProxyConfigurationNil
	}

	metrics.StartRPSTicker()

	zerolog.Ctx(ctx).Info().
		Str("udp", p.cfg.Listen.UDP).
		Str("tcp", p.cfg.Listen.TCP).
		Str("version", version.GetVersion()).
		Str("build_time", version.GetBuildTime()).
		Msg("starting DNS servers")
	metrics.SetReady(true)

	// Initialize LAN resolver if enabled
	if p.cfg.LocalZones.Enabled {
		p.initLANResolver(ctx)
	}

	// initial pipeline
	p.rebuildResolver(ctx)

	udpSrv := &dns.Server{Addr: p.cfg.Listen.UDP, Net: "udp"}
	tcpSrv := &dns.Server{Addr: p.cfg.Listen.TCP, Net: "tcp"}

	// Check ports availability by attempting to bind before ListenAndServe
	// UDP
	if c, err := (&net.ListenConfig{}).ListenPacket(ctx, "udp", p.cfg.Listen.UDP); err != nil {
		return fmt.Errorf("failed to bind UDP port %s: %w", p.cfg.Listen.UDP, err)
	} else {
		_ = c.Close()
	}
	// TCP
	if l, err := (&net.ListenConfig{}).Listen(ctx, "tcp", p.cfg.Listen.TCP); err != nil {
		return fmt.Errorf("failed to bind TCP port %s: %w", p.cfg.Listen.TCP, err)
	} else {
		_ = l.Close()
	}

	dns.HandleFunc(".", p.handleDNS(ctx))

	go func() {
		if err := udpSrv.ListenAndServe(); err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("UDP DNS server error")
		}
	}()
	go func() {
		if err := tcpSrv.ListenAndServe(); err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("TCP DNS server error")
		}
	}()

	go func() {
		<-ctx.Done()
		zerolog.Ctx(ctx).Info().Msg("shutting down DNS servers")

		if err := udpSrv.Shutdown(); err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("failed to shutdown UDP server")
		}

		if err := tcpSrv.Shutdown(); err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("failed to shutdown TCP server")
		}

		metrics.SetReady(false)
	}()

	return nil
}

func (p *Proxy) handleDNS(ctx context.Context) dns.HandlerFunc { //nolint:funcorder,funlen
	return func(w dns.ResponseWriter, r *dns.Msg) {
		// Panic recovery for DNS handler
		defer func() {
			if rec := recover(); rec != nil {
				zerolog.Ctx(ctx).Error().Interface("panic", rec).Msg("DNS handler panic recovered")
				dns.HandleFailed(w, r)
			}
		}()

		start := time.Now()

		// Extract client IP from ResponseWriter and EDNS0 options
		clientIP := extractClientIP(w, r)

		// Validate DNS message
		if r == nil {
			zerolog.Ctx(ctx).Warn().Msg("received nil DNS message")
			dns.HandleFailed(w, r)

			return
		}

		// resolve via active pipeline
		resAny := p.active.Load()

		resolver, _ := resAny.(Resolver)
		if resolver == nil {
			p.rebuildResolver(ctx)

			if res2 := p.active.Load(); res2 != nil {
				resolver, _ = res2.(Resolver)
			}
		}

		resp, usedUpstream, err := resolver.Resolve(ctx, r)
		if err != nil {
			// record error event
			if len(r.Question) > 0 {
				q := r.Question[0]

				duration := time.Since(start)
				if duration == 0 {
					duration = time.Microsecond
				}

				p.addHistory(QueryEvent{
					Name:     strings.TrimSuffix(q.Name, "."),
					QType:    q.Qtype,
					Upstream: usedUpstream,
					Duration: duration.String(),
					Status:   "error",
					Time:     time.Now(),
					ClientIP: clientIP,
				})
			}

			dns.HandleFailed(w, r)

			return
		}

		// decorators handle marking/metrics/cache

		if len(r.Question) > 0 {
			q := r.Question[0]

			duration := time.Since(start)
			if duration == 0 {
				duration = time.Microsecond
			}

			p.addHistory(QueryEvent{
				Name:     strings.TrimSpace(strings.TrimSuffix(q.Name, ".")),
				QType:    q.Qtype,
				Upstream: usedUpstream,
				Duration: duration.String(),
				Status:   "ok",
				Time:     time.Now(),
				ClientIP: clientIP,
			})
		}

		_ = w.WriteMsg(resp)
	}
}

//nolint:cyclop,funcorder,gocognit,funlen
func (p *Proxy) exchangeDoH(ctx context.Context, msg *dns.Msg, url string) (*dns.Msg, time.Duration, error) {
	if msg == nil {
		return nil, 0, errNilDNSMessageForDoH
	}

	if p.dohClient == nil {
		return nil, 0, errDoHClientNotInitialized
	}

	// Minimal DoH (RFC8484) GET with dns= base64url-encoded wireformat
	wire, err := msg.Pack()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to pack DNS message: %w", err)
	}

	b64 := base64.RawURLEncoding.EncodeToString(wire)

	u := url
	if strings.Contains(url, "?") {
		u += "&"
	} else {
		u += "?"
	}

	u += "dns=" + b64

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create DoH request: %w", err)
	}

	req.Header.Set("Accept", "application/dns-message")

	start := time.Now()

	resp, err := p.dohClient.Do(req)
	//nolint:nestif
	if err != nil {
		// Fallback: if domain resolution for DoH endpoint fails, try a pinned resolver IP via Host header
		var dnsErr *net.DNSError
		if errors.As(err, &dnsErr) || strings.Contains(err.Error(), "no such host") {
			// Try common providers
			fallback := []struct{ host, ip string }{
				{"cloudflare-dns.com", "1.1.1.1"},
				{"cloudflare-dns.com", "1.0.0.1"},
				{"dns.google", "8.8.8.8"},
				{"dns.google", "8.8.4.4"},
				{"doh.opendns.com", "208.67.222.222"},
			}
			for _, fb := range fallback {
				if !strings.Contains(url, fb.host) {
					continue
				}
				// Rebuild request to fixed IP with SNI/Host preserved
				ipURL := strings.Replace(url, fb.host, fb.ip, 1)

				req2, err2 := http.NewRequestWithContext(ctx, http.MethodGet, ipURL, nil)
				if err2 != nil {
					continue
				}

				req2.Host = fb.host
				req2.Header.Set("Accept", "application/dns-message")
				// TLS transport with SNI
				tr := &http.Transport{TLSClientConfig: &tls.Config{ServerName: fb.host, MinVersion: tls.VersionTLS13}}

				cli := &http.Client{Timeout: defaultDoHTimeout, Transport: tr}
				if resp2, err2 := cli.Do(req2); err2 == nil && resp2 != nil {
					defer func() { _ = resp2.Body.Close() }()

					if resp2.StatusCode == http.StatusOK {
						body, _ := io.ReadAll(resp2.Body)

						out := &dns.Msg{}
						if err := out.Unpack(body); err == nil {
							return out, time.Since(start), nil
						}
					}
				}
			}
		}

		return nil, 0, fmt.Errorf("DoH request failed: %w", err)
	}

	defer func() {
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("%w %d", errDoHStatus, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	out := &dns.Msg{}
	if err := out.Unpack(body); err != nil {
		return nil, 0, err
	}

	return out, time.Since(start), nil
}

// deprecated: pickUpstream no longer used with chain/failover

func parseUpstream(u string) (string, string) { //nolint:cyclop
	if u == "" {
		return protocolUDP, defaultDNS // fallback
	}

	// DoH URL passthrough
	if strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") {
		return protocolDOH, u
	}

	// Allow "doh:https://..." format coming from config aggregation
	if strings.HasPrefix(u, "doh:https://") || strings.HasPrefix(u, "doh:http://") {
		return protocolDOH, strings.TrimPrefix(u, "doh:")
	}

	// DoQ / QUIC scheme
	if strings.HasPrefix(u, "quic://") || strings.HasPrefix(u, "doq://") {
		return protocolDOQ, strings.TrimPrefix(strings.TrimPrefix(u, "doq://"), "quic://")
	}

	if strings.HasPrefix(u, "doq:") || strings.HasPrefix(u, "quic:") {
		return protocolDOQ, strings.TrimPrefix(strings.TrimPrefix(u, "doq:"), "quic:")
	}

	// DoT / TLS scheme
	if strings.HasPrefix(u, "tls://") || strings.HasPrefix(u, "dot://") {
		parts := strings.SplitN(u, "://", protocolSplitLimit)
		if len(parts) == protocolSplitLimit {
			return protocolDOT, parts[1]
		}

		return protocolDOT, u
	}

	parts := strings.SplitN(u, ":", addressSplitLimit)
	if len(parts) == addressSplitLimit {
		return parts[0], parts[1] + ":" + parts[2]
	}
	// Accept host or host:port
	if strings.Contains(u, ":") {
		if _, port, err := net.SplitHostPort(u); err == nil && port == "853" {
			return protocolDOT, u
		}

		return protocolUDP, u
	}

	return protocolUDP, u + ":53"
}

func minTTL(ttl uint32) uint32 {
	return max(ttl, defaultMinTTL)
}

// Rules returns the rule store for admin helpers.
func (p *Proxy) Rules() *RuleStore                 { return p.rules }
func (p *Proxy) GetRuleGroups() []config.RuleGroup { return p.cfg.GetRuleGroups() }
func (p *Proxy) GetConfig() *config.Config         { return p.cfg }

// PersistRules writes current rule set back to config file.
func (p *Proxy) PersistRules() error {
	p.persistMu.Lock()
	defer p.persistMu.Unlock()

	// Update rule groups with current rules
	// This is a simplified approach - in a real implementation,
	// you might want to preserve the group structure
	var allRules []config.Rule

	for _, group := range p.cfg.GetRuleGroups() {
		for _, pattern := range group.Patterns {
			allRules = append(allRules, config.Rule{
				Pattern: pattern,
				Via:     group.Via,
			})
		}
	}

	// For now, we'll put all rules in the first group or create a default group
	if len(p.cfg.RuleGroups) == 0 {
		patterns := make([]string, len(allRules))
		for i, rule := range allRules {
			patterns[i] = rule.Pattern
		}

		p.cfg.RuleGroups = []config.RuleGroup{{
			Name:     "Default",
			Via:      "default",
			Patterns: patterns,
		}}
	} else {
		patterns := make([]string, len(allRules))
		for i, rule := range allRules {
			patterns[i] = rule.Pattern
		}

		p.cfg.RuleGroups[0].Patterns = patterns
	}

	return p.cfg.Save()
}

// GetUpstreams returns upstreams helpers.
func (p *Proxy) GetUpstreams() []string {
	p.persistMu.Lock()
	defer p.persistMu.Unlock()

	return slices.Clone(p.upstreams)
}

// SetUpstreamsConfig replaces upstreams with structured configs and rebuilds pipeline.
func (p *Proxy) SetUpstreamsConfig(ctx context.Context, ups []config.UpstreamConfig) error {
	p.persistMu.Lock()
	defer p.persistMu.Unlock()
	// 1) Prepare runtime view with detected types and sane weights
	typed := make([]config.UpstreamConfig, 0, len(ups))
	for i := range ups {
		u := ups[i]
		if u.Weight <= 0 {
			u.Weight = 1
		}

		if u.Type == "" {
			u.Type = configDetectType(u.Address)
		}

		typed = append(typed, u)
	}

	// 2) Update runtime resolvers based on the typed slice
	//    Build legacy string list for resolver pipeline
	p.upstreams = nil
	for _, u := range typed {
		p.upstreams = append(p.upstreams, u.Type+":"+u.Address)
	}
	// Rebuild active resolver chain
	p.rebuildResolver(ctx)

	// 3) Persist a clean config without explicit types (omitted in YAML)
	persist := make([]config.UpstreamConfig, 0, len(ups))
	for _, u := range ups {
		// keep name/address/weight; drop Type to rely on autodetect at load
		persist = append(persist, config.UpstreamConfig{
			Name:    u.Name,
			Address: u.Address,
			Weight:  u.Weight,
			// Type intentionally left empty (omitempty)
		})
	}

	p.cfg.Upstreams = persist

	return p.cfg.Save()
}

// local shim to avoid import cycle; mirrors internal/config.detectType.
func configDetectType(addr string) string {
	if strings.HasPrefix(addr, "https://") {
		return protocolDOH
	}

	if strings.HasPrefix(addr, "quic://") || strings.HasPrefix(addr, "doq://") {
		return protocolDOQ
	}

	if strings.HasPrefix(addr, "tls://") || strings.HasPrefix(addr, "dot://") {
		return protocolDOT
	}

	if strings.HasPrefix(addr, "tcp://") {
		return protocolTCP
	}

	if strings.HasPrefix(addr, "udp://") {
		return protocolUDP
	}

	if _, port, err := net.SplitHostPort(addr); err == nil && port == "853" {
		return protocolDOT
	}

	return protocolUDP
}

// GetHosts returns hosts helpers.
func (p *Proxy) GetHosts() []config.HostOverride {
	p.persistMu.Lock()
	defer p.persistMu.Unlock()

	return slices.Clone(p.cfg.Hosts)
}

func (p *Proxy) SetHosts(ctx context.Context, hosts []config.HostOverride) error {
	p.persistMu.Lock()
	defer p.persistMu.Unlock()

	p.cfg.Hosts = hosts
	p.rebuildResolver(ctx)

	return p.cfg.Save()
}

func (p *Proxy) SetUpstreams(ctx context.Context, us []string) error { //nolint:cyclop
	p.persistMu.Lock()
	defer p.persistMu.Unlock()

	// Convert string upstreams to UpstreamConfig
	upstreams := make([]config.UpstreamConfig, 0, len(us))
	seen := map[string]struct{}{}

	for i, u := range us {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}

		if _, ok := seen[u]; ok {
			continue
		}

		// Parse upstream format: proto:host:port or https://url for DoH
		var upstream config.UpstreamConfig

		if strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "http://") {
			upstream = config.UpstreamConfig{
				Name:    fmt.Sprintf("DoH-%d", i+1),
				Address: u,
				Type:    "doh",
			}
		} else {
			// quick format check: expect proto:host:port
			parts := strings.SplitN(u, ":", addressSplitLimit)
			if len(parts) != 3 || (parts[0] != protocolUDP && parts[0] != protocolTCP) || parts[1] == "" || parts[2] == "" {
				return fmt.Errorf("%w: %s (expected proto:host:port)", errInvalidUpstream, u)
			}

			upstream = config.UpstreamConfig{
				Name:    fmt.Sprintf("%s-%s", parts[0], parts[1]),
				Address: parts[1] + ":" + parts[2],
				Type:    parts[0],
			}
		}

		seen[u] = struct{}{}

		upstreams = append(upstreams, upstream)
	}

	if len(upstreams) == 0 {
		return errAtLeastOneUpstreamRequired
	}

	p.cfg.Upstreams = upstreams
	p.upstreams = p.cfg.GetUpstreamAddresses()
	p.rebuildResolver(ctx)

	return p.cfg.Save()
}

// QueryEvent represents one DNS query record for in-memory history.
type QueryEvent struct {
	Name     string    `json:"name"`
	QType    uint16    `json:"qtype"`
	Upstream string    `json:"upstream"`
	Duration string    `json:"duration"`
	Status   string    `json:"status"`
	Time     time.Time `json:"time"`
	ClientIP string    `json:"client_ip"`
}

func (p *Proxy) addHistory(ev QueryEvent) { //nolint:funcorder
	if p.histCap <= 0 || p.histBuf == nil {
		return
	}

	p.histMu.Lock()
	defer p.histMu.Unlock()

	if p.histCap <= 0 || p.histBuf == nil {
		return
	}

	p.histBuf[p.histHead] = ev

	p.histHead++
	if p.histHead == p.histCap {
		p.histHead = 0
	}

	if p.histSize < p.histCap {
		p.histSize++
	}
}

// History returns a copy of last events (newest first).
func (p *Proxy) History() []QueryEvent {
	p.histMu.Lock()
	defer p.histMu.Unlock()

	if p.histBuf == nil || p.histCap <= 0 {
		return nil
	}

	n := p.histSize
	if n == 0 {
		return nil
	}

	out := make([]QueryEvent, n)

	idx := (p.histHead - 1 + p.histCap) % p.histCap
	for i := range n {
		if idx >= 0 && idx < len(p.histBuf) {
			out[i] = p.histBuf[idx]
		}

		if idx == 0 {
			idx = p.histCap - 1
		} else {
			idx--
		}
	}

	return out
}

// extractClientIPFromEDNS0 extracts client IP from EDNS0 Client Subnet option.
func extractClientIPFromEDNS0(r *dns.Msg) string {
	if r == nil || r.IsEdns0() == nil {
		return ""
	}

	for _, option := range r.IsEdns0().Option {
		subnet, ok := option.(*dns.EDNS0_SUBNET)
		if !ok {
			continue
		}

		ip := extractIPFromSubnet(subnet)
		if ip != "" {
			return ip
		}
	}

	return ""
}

// extractIPFromSubnet extracts IP from EDNS0 subnet option.
func extractIPFromSubnet(subnet *dns.EDNS0_SUBNET) string {
	if subnet.Family == 1 && len(subnet.Address) >= 4 { // IPv4
		if !subnet.Address.IsUnspecified() {
			return subnet.Address.String()
		}
	} else if subnet.Family == 2 && len(subnet.Address) >= 16 { // IPv6
		if !subnet.Address.IsUnspecified() {
			return subnet.Address.String()
		}
	}

	return ""
}

// extractClientIPFromRemoteAddr extracts client IP from RemoteAddr.
func extractClientIPFromRemoteAddr(w dns.ResponseWriter) string {
	addr := w.RemoteAddr()
	if addr == nil {
		return "unknown"
	}

	if host, _, err := net.SplitHostPort(addr.String()); err == nil {
		return host
	}

	return addr.String()
}

// extractClientIP extracts the real client IP from DNS request.
// Uses EDNS0 Client Subnet if available, otherwise falls back to RemoteAddr.
func extractClientIP(w dns.ResponseWriter, r *dns.Msg) string {
	// Try EDNS0 Client Subnet first (standard way to get real client IP)
	if ip := extractClientIPFromEDNS0(r); ip != "" {
		return ip
	}

	// Fallback to RemoteAddr (direct connection or no EDNS0)
	return extractClientIPFromRemoteAddr(w)
}

// initLANResolver initializes the LAN resolver and file watcher.
func (p *Proxy) initLANResolver(ctx context.Context) {
	logger := zerolog.Ctx(ctx)

	// Set default paths if not specified
	cfg := &p.cfg.LocalZones
	if cfg.UCIPath == "" {
		cfg.UCIPath = "/etc/config/dhcp"
	}
	if cfg.ResolvPath == "" {
		cfg.ResolvPath = "/tmp/resolv.conf.auto"
	}
	if cfg.LeasesPath == "" {
		cfg.LeasesPath = "/tmp/dhcp.leases"
	}

	// Detect local zones
	zones := localzone.DetectLocalZones(cfg.LocalZones, cfg.UCIPath, cfg.ResolvPath)
	logger.Info().Strs("zones", zones).Msg("detected local DNS zones")

	// Parse initial leases
	leases, err := lanresolver.ParseLeases(cfg.LeasesPath)
	if err != nil {
		logger.Warn().Err(err).Str("path", cfg.LeasesPath).Msg("failed to parse DHCP leases, will retry on file changes")
		leases = []lanresolver.Lease{}
	} else {
		logger.Info().Int("count", len(leases)).Msg("loaded DHCP leases")
	}

	// Create LAN resolver (will be added to chain in rebuildResolver)
	p.lanResolver = lanresolver.NewResolver(nil, zones, p.cfg)
	p.lanResolver.UpdateLeases(leases, zones)

	// Set up file watcher
	watcher, err := localzone.NewWatcher()
	if err != nil {
		logger.Warn().Err(err).Msg("failed to create file watcher for LAN resolver")
		return
	}
	p.lanWatcher = watcher

	// Callback to reload leases and zones
	reloadFn := func() {
		// Reload zones
		newZones := localzone.DetectLocalZones(cfg.LocalZones, cfg.UCIPath, cfg.ResolvPath)

		// Reload leases
		newLeases, err := lanresolver.ParseLeases(cfg.LeasesPath)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to reload DHCP leases")
			// Keep old leases on error
			newLeases = p.lanResolver.GetLeases()
		}

		logger.Debug().
			Strs("zones", newZones).
			Int("leases", len(newLeases)).
			Msg("reloaded local DNS data")

		p.lanResolver.UpdateLeases(newLeases, newZones)
	}

	watcher.OnChange(reloadFn)

	// Start watching files
	filesToWatch := []string{cfg.LeasesPath, cfg.UCIPath, cfg.ResolvPath}
	watcher.Watch(ctx, filesToWatch)

	logger.Info().Strs("files", filesToWatch).Msg("watching files for local DNS changes")
}

// GetLANResolver returns the LAN resolver if initialized.
func (p *Proxy) GetLANResolver() *lanresolver.LANResolver {
	return p.lanResolver
}

// ensure import usage.
var _ = net.IP{}
