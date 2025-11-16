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
	// Managers for thread-safe operations
	upstreams UpstreamsManager
	hosts     HostsManager
	cache     CacheManager
	history   HistoryManager
	rules     RulesManager
	config    ConfigManager

	// Core components
	backend      firewall.Backend
	active       atomic.Value       // Resolver
	asyncMarkRes *AsyncMarkResolver // Reference to async mark resolver for cleanup

	// DNS clients
	dnsUDP    *dns.Client
	dnsTCP    *dns.Client
	dohClient *http.Client
}

// ResolverActive returns the current active resolver atomically.
//
//nolint:ireturn
func (p *Proxy) ResolverActive() Resolver {
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
		backend:   backend,
		dnsUDP:    &dns.Client{Net: "udp", Timeout: defaultDNSTimeout},
		dnsTCP:    &dns.Client{Net: "tcp", Timeout: defaultDNSTimeout},
		dohClient: &http.Client{Timeout: defaultDoHTimeout},
	}

	// Initialize managers
	p.config = newConfigManager(cfg)
	p.upstreams = newUpstreamsManager(p, cfg.Upstreams)
	p.hosts = newHostsManager(cfg)
	p.history = newHistoryManager(capacity)
	p.rules = newRulesManager(NewRuleStore(cfg.GetAllRules()), cfg.RuleGroups)

	// Initialize cache if enabled
	if cfg.Cache.Enabled {
		cache := NewCachedResolverWithSize(
			nil, // Will be set in rebuildResolver
			cfg.Cache.MaxEntries,
			cfg.Cache.MaxSizeMB,
			cfg.Cache.MinTTLSeconds,
			cfg.Cache.MaxTTLSeconds,
		)
		p.cache = newCacheManager(cache)
	}

	// Initialize the resolver pipeline
	p.rebuildResolver(context.Background())

	return p
}

func (p *Proxy) Start(ctx context.Context) error {
	cfg := p.config.GetConfig()
	if cfg == nil {
		return errProxyConfigurationNil
	}

	metrics.StartRPSTicker()

	zerolog.Ctx(ctx).Info().
		Str("udp", cfg.Listen.UDP).
		Str("tcp", cfg.Listen.TCP).
		Str("version", version.GetVersion()).
		Str("build_time", version.GetBuildTime()).
		Msg("starting DNS servers")
	metrics.SetReady(true)
	// initial pipeline
	p.rebuildResolver(ctx)

	udpSrv := &dns.Server{Addr: cfg.Listen.UDP, Net: "udp"}
	tcpSrv := &dns.Server{Addr: cfg.Listen.TCP, Net: "tcp"}

	// Check ports availability by attempting to bind before ListenAndServe
	// UDP
	if c, err := (&net.ListenConfig{}).ListenPacket(ctx, "udp", cfg.Listen.UDP); err != nil {
		return fmt.Errorf("failed to bind UDP port %s: %w", cfg.Listen.UDP, err)
	} else {
		_ = c.Close()
	}
	// TCP
	if l, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.Listen.TCP); err != nil {
		return fmt.Errorf("failed to bind TCP port %s: %w", cfg.Listen.TCP, err)
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

		// Stop async mark resolver
		if p.asyncMarkRes != nil {
			p.asyncMarkRes.Stop()
		}

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

func (p *Proxy) handleDNS(ctx context.Context) dns.HandlerFunc { //nolint:funcorder,funlen,cyclop
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

				logger := zerolog.Ctx(ctx).With().
					Str("query", strings.TrimSuffix(q.Name, ".")).
					Uint16("qtype", q.Qtype).
					Str("upstream", usedUpstream).
					Dur("duration", duration).
					Str("client_ip", clientIP).
					Logger()

				logger.Error().Err(err).Msg("DNS resolution failed")

				p.history.AddEvent(QueryEvent{
					Name:     strings.TrimSuffix(q.Name, "."),
					QType:    q.Qtype,
					Upstream: usedUpstream,
					Duration: duration.String(),
					Status:   "error",
					Time:     time.Now(),
					ClientIP: clientIP,
				})
			} else {
				zerolog.Ctx(ctx).Error().Err(err).Str("upstream", usedUpstream).Msg("DNS resolution failed (no question)")
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

			queryName := strings.TrimSpace(strings.TrimSuffix(q.Name, "."))

			answersCount := 0
			if resp != nil {
				answersCount = len(resp.Answer)
			}

			// Log successful resolution at debug level to avoid spam
			zerolog.Ctx(ctx).Debug().
				Str("query", queryName).
				Uint16("qtype", q.Qtype).
				Str("upstream", usedUpstream).
				Dur("duration", duration).
				Int("answers", answersCount).
				Str("client_ip", clientIP).
				Msg("DNS resolution successful")

			p.history.AddEvent(QueryEvent{
				Name:     queryName,
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

	// UDP/TCP scheme support (udp://host:port, tcp://host:port)
	if after, ok := strings.CutPrefix(u, "udp://"); ok {
		return protocolUDP, after
	}

	if after, ok := strings.CutPrefix(u, "tcp://"); ok {
		return protocolTCP, after
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
func (p *Proxy) Rules() *RuleStore                 { return p.rules.GetRules() }
func (p *Proxy) GetRuleGroups() []config.RuleGroup { return p.rules.GetRuleGroups() }
func (p *Proxy) GetConfig() *config.Config         { return p.config.GetConfig() }

// Cache returns the cache resolver for admin operations.
func (p *Proxy) Cache() *CachedResolver {
	if p.cache != nil {
		return p.cache.GetCache()
	}

	return nil
}

// PersistRules writes current rule set back to config file.
func (p *Proxy) PersistRules() error {
	// Update rule groups with current rules
	// This is a simplified approach - in a real implementation,
	// you might want to preserve the group structure
	var allRules []config.Rule

	cfg := p.config.GetConfig()
	for _, group := range cfg.GetRuleGroups() {
		for _, pattern := range group.Patterns {
			allRules = append(allRules, config.Rule{
				Pattern: pattern,
				Via:     group.Via,
			})
		}
	}

	// For now, we'll put all rules in the first group or create a default group
	if len(cfg.RuleGroups) == 0 {
		patterns := make([]string, len(allRules))
		for i, rule := range allRules {
			patterns[i] = rule.Pattern
		}

		cfg.RuleGroups = []config.RuleGroup{{
			Name:     "Default",
			Via:      "default",
			Patterns: patterns,
		}}
	} else {
		patterns := make([]string, len(allRules))
		for i, rule := range allRules {
			patterns[i] = rule.Pattern
		}

		cfg.RuleGroups[0].Patterns = patterns
	}

	if err := p.config.SaveConfig(); err != nil {
		// Note: ctx might not be available here, but we can still log
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// GetUpstreams returns upstreams helpers.
func (p *Proxy) GetUpstreams() []string {
	return p.upstreams.GetUpstreamAddresses()
}

// SetUpstreamsConfig replaces upstreams with structured configs and rebuilds pipeline.
func (p *Proxy) SetUpstreamsConfig(ctx context.Context, ups []config.UpstreamConfig) error {
	logger := zerolog.Ctx(ctx).With().Int("upstreams_count", len(ups)).Logger()

	logger.Info().Msg("updating upstreams configuration")

	// 1) Validate all upstreams before processing (critical for preventing invalid state)
	// Validate each upstream individually to catch errors early
	for i, u := range ups {
		if u.Name == "" {
			return fmt.Errorf("upstream #%d: name cannot be empty", i+1)
		}

		if u.Address == "" {
			return fmt.Errorf("upstream '%s': address cannot be empty", u.Name)
		}

		if u.Weight < 0 {
			return fmt.Errorf("upstream '%s': weight cannot be negative (got %d)", u.Name, u.Weight)
		}
	}

	// Ensure at least one upstream
	if len(ups) == 0 {
		return errors.New("at least one upstream is required")
	}

	// 2) Prepare runtime view with detected types and sane weights
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

	// 3) Update upstreams manager atomically (thread-safe, with validation already done)
	if err := p.upstreams.SetUpstreams(typed); err != nil {
		logger.Error().Err(err).Msg("failed to update upstreams in manager")

		return fmt.Errorf("failed to update upstreams: %w", err)
	}

	logger.Debug().Int("valid_upstreams", len(typed)).Msg("upstreams updated successfully in manager (in-memory)")

	// 4) Rebuild active resolver chain (this is fast, happens in memory)
	// Note: rebuildResolver can fail, but upstreams are already updated in memory
	// This is acceptable because the resolver will be rebuilt on next request if needed
	p.rebuildResolver(ctx)

	logger.Debug().Msg("resolver pipeline rebuilt successfully")

	// 5) Persist a clean config without explicit types (omitted in YAML)
	// Normalize addresses: remove udp:// and tcp:// prefixes, keep only host:port
	persist := make([]config.UpstreamConfig, 0, len(ups))
	for _, u := range ups {
		normalizedAddr := u.Address
		// Remove scheme prefixes for UDP/TCP to normalize storage format
		if after, ok := strings.CutPrefix(normalizedAddr, "udp://"); ok {
			normalizedAddr = after
		} else if after, ok := strings.CutPrefix(normalizedAddr, "tcp://"); ok {
			normalizedAddr = after
		}
		// Keep name/address/weight; drop Type to rely on autodetect at load
		persist = append(persist, config.UpstreamConfig{
			Name:    u.Name,
			Address: normalizedAddr,
			Weight:  u.Weight,
			// Type intentionally left empty (omitempty)
		})
	}

	// 6) Save configuration asynchronously to avoid blocking DNS proxy
	// This prevents service disruption if disk I/O is slow
	go func() {
		saveLogger := logger.With().Str("operation", "async_save").Logger()

		cfg := p.config.GetConfig()
		cfg.Upstreams = persist

		if err := p.config.SaveConfig(); err != nil {
			saveLogger.Error().
				Err(err).
				Msg("failed to save config after updating upstreams (async save failed)")

			// Log error but don't fail the request - upstreams are already updated in memory
			// The next successful save will persist the changes
			return
		}

		saveLogger.Info().Msg("upstreams configuration saved successfully (async)")
	}()

	logger.Info().Msg("upstreams configuration updated successfully (saved asynchronously)")

	return nil
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
	return p.hosts.GetHosts()
}

func (p *Proxy) SetHosts(ctx context.Context, hosts []config.HostOverride) error {
	logger := zerolog.Ctx(ctx).With().Int("hosts_count", len(hosts)).Logger()

	logger.Info().Msg("updating hosts configuration")

	// Validate hosts count before processing
	const maxHostsPerRequest = 1000
	if len(hosts) > maxHostsPerRequest {
		return fmt.Errorf("too many hosts (max %d, got %d)", maxHostsPerRequest, len(hosts))
	}

	// Update hosts in place (thread-safe, with validation)
	// This validates all hosts before updating, preventing invalid state
	if err := p.hosts.UpdateHostsInPlace(hosts); err != nil {
		logger.Error().Err(err).Msg("failed to update hosts in place (validation failed)")

		return fmt.Errorf("validation failed: %w", err)
	}

	logger.Debug().Msg("hosts updated successfully in manager (in-memory)")

	// Note: rebuildResolver() is no longer needed because HostsResolver is now dynamic
	// and reads hosts from manager on each request. This improves performance by avoiding
	// full pipeline rebuild on every hosts update.
	// If you need to rebuild for other reasons, you can still call rebuildResolver(ctx)

	// Save configuration asynchronously to avoid blocking DNS proxy
	// This prevents service disruption if disk I/O is slow
	go func() {
		saveLogger := logger.With().Str("operation", "async_save").Logger()

		if err := p.config.SaveConfig(); err != nil {
			saveLogger.Error().
				Err(err).
				Msg("failed to save config after updating hosts (async save failed)")

			// Log error but don't fail the request - hosts are already updated in memory
			// The next successful save will persist the changes
			return
		}

		saveLogger.Info().Msg("hosts configuration saved successfully (async)")
	}()

	logger.Info().Msg("hosts configuration updated successfully (saved asynchronously)")

	return nil
}

func (p *Proxy) SetUpstreams(ctx context.Context, us []string) error { //nolint:cyclop,funlen
	logger := zerolog.Ctx(ctx).With().Int("upstreams_count", len(us)).Logger()

	logger.Info().Msg("updating upstreams configuration")

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

	// Update upstreams manager
	if err := p.upstreams.SetUpstreams(upstreams); err != nil {
		logger.Error().Err(err).Msg("failed to update upstreams in manager")

		return err
	}

	logger.Debug().Int("valid_upstreams", len(upstreams)).Msg("upstreams updated successfully in manager")

	// Rebuild active resolver chain
	p.rebuildResolver(ctx)

	logger.Info().Msg("upstreams configuration updated and pipeline rebuilt successfully")

	// Update config
	cfg := p.config.GetConfig()
	cfg.Upstreams = upstreams

	if err := p.config.SaveConfig(); err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("failed to save config after updating upstreams")

		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
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

// History returns a copy of last events (newest first).
func (p *Proxy) History() []QueryEvent {
	return p.history.GetHistory(0) // 0 means return all
}

// HistoryPaginated returns paginated history with offset and limit.
func (p *Proxy) HistoryPaginated(offset, limit int) []QueryEvent {
	return p.history.GetHistoryPaginated(offset, limit)
}

// HistorySize returns the total number of history events.
func (p *Proxy) HistorySize() int {
	return p.history.GetHistorySize()
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

// ensure import usage.
var _ = net.IP{}
