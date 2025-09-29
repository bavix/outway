package adminhttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/unrolled/secure"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/dnsproxy"
	"github.com/bavix/outway/internal/metrics"
	"github.com/bavix/outway/internal/updater"
	"github.com/bavix/outway/internal/version"
	"github.com/bavix/outway/ui"
)

var (
	errNameViaPatternsRequired = errors.New("name, via and patterns are required")
	errRuleGroupExists         = errors.New("rule group already exists")
	errUpstreamsRequired       = errors.New("upstreams required")
	errRuleGroupNotFound       = errors.New("rule group not found")
	errNameRequired            = errors.New("name required")
	errUnsupportedType         = errors.New("unsupported type")
	errResolverNotReady        = errors.New("resolver not ready")
)

const (
	defaultReadHeaderTimeout         = 5 * time.Second
	defaultIdleTimeout               = 10 * time.Second
	defaultWriteTimeout              = 15 * time.Second
	defaultShutdownTimeout           = 5 * time.Second
	defaultHealthCheckInterval       = 5 * time.Second
	defaultWebSocketReadLimit        = 1024
	defaultWebSocketTimeout          = 60 * time.Second
	defaultWebSocketPingInterval     = 30 * time.Second
	defaultWebSocketPingTimeout      = 5 * time.Second
	defaultWebSocketWriteTimeout     = 5 * time.Second
	defaultBadGatewayStatus          = 502
	defaultBadRequestStatus          = 400
	defaultInternalServerErrorStatus = 500
)

type Server struct {
	addr      string
	mux       *mux.Router
	proxy     *dnsproxy.Proxy
	updater   *updater.Updater
	wsMu      sync.RWMutex // protects conns map (read for iteration, write for add/remove)
	wsWriteMu sync.Mutex   // protects WebSocket writes (WriteJSON is not a "read" operation)
	conns     map[*websocket.Conn]struct{}
	startTime time.Time
	version   string
	buildTime string
	adminPort int
	dnsPort   string
}

func NewServer(addr string, proxy *dnsproxy.Proxy) *Server {
	s := &Server{
		addr:  addr,
		mux:   mux.NewRouter(),
		proxy: proxy,
		updater: func() *updater.Updater {
			u, err := updater.New(updater.Config{
				Owner:          "bavix",
				Repo:           "outway",
				CurrentVersion: version.GetVersion(),
				BinaryName:     "outway",
			})
			if err != nil {
				panic(err) // This should not happen with valid config
			}

			return u
		}(),
		conns:     make(map[*websocket.Conn]struct{}),
		startTime: time.Now(),
		version:   version.GetVersion(),
		buildTime: version.GetBuildTime(),
	}

	// Derive ports from inputs and loaded config
	if _, port, err := net.SplitHostPort(addr); err == nil {
		s.adminPort, _ = net.DefaultResolver.LookupPort(context.Background(), "tcp", port)
	}

	if cfg := proxy.GetConfig(); cfg != nil {
		switch {
		case cfg.Listen.UDP != "":
			s.dnsPort = cfg.Listen.UDP
		case cfg.Listen.TCP != "":
			s.dnsPort = cfg.Listen.TCP
		default:
			s.dnsPort = ":53"
		}
	}

	s.routes()

	return s
}

func NewServerWithConfig(httpConfig *config.HTTPConfig, proxy *dnsproxy.Proxy) *Server {
	s := &Server{
		addr:  httpConfig.Listen,
		mux:   mux.NewRouter(),
		proxy: proxy,
		updater: func() *updater.Updater {
			u, err := updater.New(updater.Config{
				Owner:          "bavix",
				Repo:           "outway",
				CurrentVersion: version.GetVersion(),
				BinaryName:     "outway",
			})
			if err != nil {
				panic(err)
			}

			return u
		}(),
		conns:     make(map[*websocket.Conn]struct{}),
		startTime: time.Now(),
		version:   version.GetVersion(),
		buildTime: version.GetBuildTime(),
	}

	s.routes()
	// Fill ports from provided config and proxy config
	if _, port, err := net.SplitHostPort(httpConfig.Listen); err == nil {
		s.adminPort, _ = net.DefaultResolver.LookupPort(context.Background(), "tcp", port)
	}

	if cfg := proxy.GetConfig(); cfg != nil {
		switch {
		case cfg.Listen.UDP != "":
			s.dnsPort = cfg.Listen.UDP
		case cfg.Listen.TCP != "":
			s.dnsPort = cfg.Listen.TCP
		default:
			s.dnsPort = ":53"
		}
	}

	return s
}

// SetVersion allows cmd layer to propagate version/build time.
func (s *Server) SetVersion(ver, build string) {
	if ver != "" {
		s.version = ver
	}

	if build != "" {
		s.buildTime = build
	}
}

func (s *Server) Start(ctx context.Context) error {
	// Fast-fail if port is occupied
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", s.addr)
	if err != nil {
		return err
	}

	_ = ln.Close()

	// Build middleware chain
	handler := s.buildMiddlewareChain(ctx)

	// Create server with middleware and graceful shutdown
	srv := s.createServer(ctx, handler)

	zerolog.Ctx(ctx).Info().
		Str("addr", s.addr).
		Str("version", s.version).
		Str("build_time", s.buildTime).
		Msg("http listen")

	go func() { _ = srv.ListenAndServe() }()

	// periodic WS broadcasts
	go func() {
		ticker := time.NewTicker(defaultHealthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.broadcast(map[string]any{"type": "stats", "data": s.collectStats()})
				s.broadcast(map[string]any{"type": "history", "data": s.proxy.History()})
				// Overview snapshot (lightweight)
				groups := s.proxy.GetRuleGroups()
				ups := s.proxy.GetConfig().Upstreams
				hist := s.proxy.History()

				var rulesTotal int
				for _, g := range groups {
					rulesTotal += len(g.Patterns)
				}

				s.broadcast(map[string]any{"type": "overview", "data": map[string]any{
					"rules_total":     rulesTotal,
					"upstreams_total": len(ups),
					"history_total":   len(hist),
					"uptime":          time.Since(s.startTime).Round(time.Second).String(),
				}})
			}
		}
	}()

	return nil
}

func (s *Server) routes() {
	// API v1 routes
	api := s.mux.PathPrefix("/api/v1").Subrouter()

	// Rule groups management
	api.HandleFunc("/rule-groups", s.handleRuleGroups).Methods("GET", "POST")
	api.HandleFunc("/rule-groups/{name}", s.handleRuleGroup).Methods("GET", "PUT", "DELETE")

	// Upstreams management
	api.HandleFunc("/upstreams", s.handleUpstreams).Methods("GET", "POST", "PUT", "DELETE")

	// Hosts management
	api.HandleFunc("/hosts", s.handleHosts).Methods("GET", "PUT")

	// Statistics and monitoring
	api.HandleFunc("/stats", s.handleStats).Methods("GET")
	api.HandleFunc("/history", s.handleHistory).Methods("GET")
	api.HandleFunc("/info", s.handleInfo).Methods("GET")
	api.HandleFunc("/overview", s.handleOverview).Methods("GET")

	// DNS test resolve
	api.HandleFunc("/resolve", s.handleResolveTest).Methods("GET")

	// Update management
	api.HandleFunc("/update/check", s.handleUpdateCheck).Methods("GET")
	api.HandleFunc("/update/download", s.handleUpdateDownload).Methods("POST")
	api.HandleFunc("/update/install", s.handleUpdateInstall).Methods("POST")
	api.HandleFunc("/update/status", s.handleUpdateStatus).Methods("GET")

	// Health check
	s.mux.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Metrics
	s.mux.Handle("/metrics", promhttp.Handler())

	// Static files and SPA fallback
	if staticFS, err := fs.Sub(ui.Assets, "dist/static"); err == nil {
		s.mux.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	}

	s.mux.PathPrefix("/").HandlerFunc(serveIndex)
}

type ruleGroupDTO struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Via         string   `json:"via"`
	Patterns    []string `json:"patterns"`
	PinTTL      bool     `json:"pin_ttl"`
}

type rulesResponse struct {
	RuleGroups []ruleGroupDTO `json:"rule_groups"`
}

type upstreamsResponse struct {
	Upstreams []config.UpstreamConfig `json:"upstreams"`
}

type serverInfoDTO struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	AdminPort int    `json:"admin_port"`
	DNSPort   string `json:"dns_port"`
	Uptime    string `json:"uptime"`
	BuildTime string `json:"build_time,omitempty"`
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	// Serve index.html for SPA routing
	data, err := ui.Assets.ReadFile("dist/index.html")
	if err != nil {
		http.Error(w, "ui not found", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) handleRuleGroups(w http.ResponseWriter, r *http.Request) { //nolint:cyclop,funlen
	switch r.Method {
	case http.MethodGet:
		var ruleGroups []ruleGroupDTO
		for _, group := range s.proxy.GetRuleGroups() {
			ruleGroups = append(ruleGroups, ruleGroupDTO{
				Name:        group.Name,
				Description: group.Description,
				Via:         group.Via,
				Patterns:    group.Patterns,
				PinTTL:      group.PinTTL,
			})
		}

		render.Status(r, http.StatusOK)
		render.JSON(w, r, rulesResponse{RuleGroups: ruleGroups})
	case http.MethodPost:
		// Create a new rule group
		var in ruleGroupDTO
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": err.Error()})

			return
		}

		if in.Name == "" || in.Via == "" || len(in.Patterns) == 0 {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": errNameViaPatternsRequired.Error()})

			return
		}
		// Check duplicates
		for _, g := range s.proxy.GetConfig().GetRuleGroups() {
			if g.Name == in.Name {
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, map[string]string{"error": errRuleGroupExists.Error()})

				return
			}
		}
		// Append to config
		cfg := s.proxy.GetConfig()
		cfg.RuleGroups = append(cfg.RuleGroups, config.RuleGroup{
			Name:        in.Name,
			Description: in.Description,
			Via:         in.Via,
			Patterns:    in.Patterns,
			PinTTL:      in.PinTTL,
		})
		// Update runtime rules store
		for _, p := range in.Patterns {
			s.proxy.Rules().Upsert(config.Rule{Pattern: p, Via: in.Via, PinTTL: in.PinTTL})
		}

		if err := cfg.Save(); err != nil {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{"error": err.Error()})

			return
		}

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, in)
		// Broadcast updated groups
		var outGroups []ruleGroupDTO
		for _, group := range cfg.GetRuleGroups() {
			outGroups = append(outGroups, ruleGroupDTO{
				Name:        group.Name,
				Description: group.Description,
				Via:         group.Via,
				Patterns:    group.Patterns,
				PinTTL:      group.PinTTL,
			})
		}

		s.broadcast(map[string]any{"type": "rule_groups", "data": outGroups})
	case http.MethodDelete:
		// Rule group deletion not implemented yet
		w.WriteHeader(http.StatusNotImplemented)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	st, err := metrics.GatherStats(metrics.Service())
	if err != nil {
		render.Status(r, defaultInternalServerErrorStatus)
		render.JSON(w, r, map[string]string{"error": err.Error()})

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, st)
}

type historyResponse struct {
	Events []dnsproxy.QueryEvent `json:"events"`
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	h := s.proxy.History()
	events := make([]dnsproxy.QueryEvent, len(h))
	copy(events, h)

	render.Status(r, http.StatusOK)
	render.JSON(w, r, historyResponse{Events: events})
}

func (s *Server) handleUpstreams(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return structured upstreams from config (type may be empty -> autodetected)
		// Normalize addresses to include scheme for UI consistency (align with WS snapshot)
		{
			ups := s.proxy.GetConfig().Upstreams

			norm := make([]config.UpstreamConfig, 0, len(ups))
			for _, u := range ups {
				a := u.Address
				if !strings.Contains(a, "://") && u.Type != "doh" {
					a = u.Type + "://" + a
				}

				norm = append(norm, config.UpstreamConfig{Name: u.Name, Address: a, Weight: u.Weight, Type: u.Type})
			}

			render.Status(r, http.StatusOK)
			render.JSON(w, r, upstreamsResponse{Upstreams: norm})
		}
	case http.MethodPost:
		var in struct {
			Upstreams []config.UpstreamConfig `json:"upstreams"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			render.Status(r, defaultBadRequestStatus)
			render.JSON(w, r, map[string]string{"error": err.Error()})

			return
		}

		if len(in.Upstreams) == 0 {
			render.Status(r, defaultBadRequestStatus)
			render.JSON(w, r, map[string]string{"error": errUpstreamsRequired.Error()})

			return
		}

		if err := s.proxy.SetUpstreamsConfig(r.Context(), in.Upstreams); err != nil {
			render.Status(r, defaultInternalServerErrorStatus)
			render.JSON(w, r, map[string]string{"error": err.Error()})

			return
		}

		w.WriteHeader(http.StatusNoContent)
		s.broadcast(map[string]any{"type": "upstreams", "data": s.proxy.GetConfig().Upstreams})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleHosts returns or updates hosts overrides.
func (s *Server) handleHosts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]any{"hosts": s.proxy.GetHosts()})
	case http.MethodPut:
		var in struct {
			Hosts []config.HostOverride `json:"hosts"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			render.Status(r, defaultBadRequestStatus)
			render.JSON(w, r, map[string]string{"error": err.Error()})

			return
		}

		if err := s.proxy.SetHosts(r.Context(), in.Hosts); err != nil {
			render.Status(r, defaultInternalServerErrorStatus)
			render.JSON(w, r, map[string]string{"error": err.Error()})

			return
		}

		w.WriteHeader(http.StatusNoContent)
		s.broadcast(map[string]any{"type": "hosts", "data": s.proxy.GetHosts()})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }} //nolint:gochecknoglobals // websocket upgrader

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) { //nolint:cyclop,funlen
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Log the error but don't use http.Error as it conflicts with WebSocket upgrade
		zerolog.Ctx(r.Context()).Err(err).Msg("WebSocket upgrade failed")

		return
	}

	s.wsMu.Lock()
	s.conns[conn] = struct{}{}
	s.wsMu.Unlock()

	// Send initial snapshot
	s.sendJSON(conn, map[string]any{"type": "stats", "data": s.collectStats()})
	s.sendJSON(conn, map[string]any{"type": "history", "data": s.proxy.History()})

	// Check for updates when WebSocket connects
	s.checkAndNotifyUpdates(conn, r.Context())
	// initial overview
	{
		groups := s.proxy.GetRuleGroups()
		ups := s.proxy.GetConfig().Upstreams

		var rulesTotal int
		for _, g := range groups {
			rulesTotal += len(g.Patterns)
		}

		s.sendJSON(conn, map[string]any{"type": "overview", "data": map[string]any{
			"rules_total":     rulesTotal,
			"upstreams_total": len(ups),
		}})
	}
	// Send hosts snapshot too
	s.sendJSON(conn, map[string]any{"type": "hosts", "data": s.proxy.GetHosts()})

	// Convert rule groups to new format
	groups := s.proxy.GetRuleGroups()

	ruleGroups := make([]ruleGroupDTO, 0, len(groups))
	for _, group := range groups {
		ruleGroups = append(ruleGroups, ruleGroupDTO{
			Name:        group.Name,
			Description: group.Description,
			Via:         group.Via,
			Patterns:    group.Patterns,
			PinTTL:      group.PinTTL,
		})
	}

	s.sendJSON(conn, map[string]any{"type": "rule_groups", "data": ruleGroups})
	// Ensure addresses in snapshot include scheme for UI consistency
	{
		ups := s.proxy.GetConfig().Upstreams

		norm := make([]config.UpstreamConfig, 0, len(ups))
		for _, u := range ups {
			a := u.Address
			if !strings.Contains(a, "://") && u.Type != "doh" {
				a = u.Type + "://" + a
			}

			norm = append(norm, config.UpstreamConfig{Name: u.Name, Address: a, Weight: u.Weight, Type: u.Type})
		}

		s.sendJSON(conn, map[string]any{"type": "upstreams", "data": norm})
	}

	// Configure connection
	conn.SetReadLimit(defaultWebSocketReadLimit)
	_ = conn.SetReadDeadline(time.Now().Add(defaultWebSocketTimeout))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(defaultWebSocketTimeout))

		return nil
	})

	// Start ping ticker
	go func(c *websocket.Conn) {
		ticker := time.NewTicker(defaultWebSocketPingInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := c.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(defaultWebSocketPingTimeout)); err != nil {
				break
			}
		}
	}(conn)

	// Handle incoming messages
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}

	// Cleanup
	s.wsMu.Lock()
	delete(s.conns, conn)
	s.wsMu.Unlock()

	_ = conn.Close()
}

func (s *Server) collectStats() metrics.Stats {
	st, _ := metrics.GatherStats(metrics.Service())

	return st
}

func (s *Server) sendJSON(c *websocket.Conn, v any) {
	s.wsWriteMu.Lock()
	defer s.wsWriteMu.Unlock()

	_ = c.SetWriteDeadline(time.Now().Add(defaultWebSocketWriteTimeout))
	_ = c.WriteJSON(v)
}

func (s *Server) broadcast(v any) {
	s.wsMu.RLock()
	defer s.wsMu.RUnlock()

	var wg sync.WaitGroup
	for c := range s.conns {
		wg.Go(func() {
			s.sendJSON(c, v)
		})
	}

	wg.Wait()
}

// handleRuleGroup handles individual rule group operations.
func (s *Server) handleRuleGroup(w http.ResponseWriter, r *http.Request) { //nolint:gocognit,cyclop,funlen
	vars := mux.Vars(r)
	name := vars["name"]

	switch r.Method {
	case http.MethodGet:
		// Get specific rule group
		groups := s.proxy.GetRuleGroups()
		for _, group := range groups {
			if group.Name == name {
				render.Status(r, http.StatusOK)
				render.JSON(w, r, ruleGroupDTO{
					Name:        group.Name,
					Description: group.Description,
					Via:         group.Via,
					Patterns:    group.Patterns,
					PinTTL:      group.PinTTL,
				})

				return
			}
		}

		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, map[string]string{"error": errRuleGroupNotFound.Error()})

	case http.MethodPut:
		// Update rule group
		var in ruleGroupDTO
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, map[string]string{"error": err.Error()})

			return
		}

		cfg := s.proxy.GetConfig()
		updated := false

		for i, g := range cfg.RuleGroups {
			if g.Name == name {
				// remove old patterns from runtime store
				for _, p := range g.Patterns {
					s.proxy.Rules().Delete(p)
				}
				// update config
				cfg.RuleGroups[i] = config.RuleGroup{
					Name:        name,
					Description: in.Description,
					Via:         in.Via,
					Patterns:    in.Patterns,
					PinTTL:      in.PinTTL,
				}
				// add new patterns to runtime store
				for _, p := range in.Patterns {
					s.proxy.Rules().Upsert(config.Rule{Pattern: p, Via: in.Via, PinTTL: in.PinTTL})
				}

				updated = true

				break
			}
		}

		if !updated {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": errRuleGroupNotFound.Error()})

			return
		}

		if err := cfg.Save(); err != nil {
			render.Status(r, defaultInternalServerErrorStatus)
			render.JSON(w, r, map[string]string{"error": err.Error()})

			return
		}

		w.WriteHeader(http.StatusNoContent)
		// broadcast
		var outGroups []ruleGroupDTO
		for _, group := range cfg.GetRuleGroups() {
			outGroups = append(outGroups, ruleGroupDTO{
				Name:        group.Name,
				Description: group.Description,
				Via:         group.Via,
				Patterns:    group.Patterns,
				PinTTL:      group.PinTTL,
			})
		}

		s.broadcast(map[string]any{"type": "rule_groups", "data": outGroups})

	case http.MethodDelete:
		// Delete rule group
		cfg := s.proxy.GetConfig()
		idx := -1

		var g config.RuleGroup

		for i, rg := range cfg.RuleGroups {
			if rg.Name == name {
				idx = i
				g = rg

				break
			}
		}

		if idx == -1 {
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, map[string]string{"error": errRuleGroupNotFound.Error()})

			return
		}
		// remove patterns from runtime store
		for _, p := range g.Patterns {
			s.proxy.Rules().Delete(p)
		}
		// remove from config
		cfg.RuleGroups = append(cfg.RuleGroups[:idx], cfg.RuleGroups[idx+1:]...)
		if err := cfg.Save(); err != nil {
			render.Status(r, defaultInternalServerErrorStatus)
			render.JSON(w, r, map[string]string{"error": err.Error()})

			return
		}

		w.WriteHeader(http.StatusNoContent)
		// broadcast
		var outGroups []ruleGroupDTO
		for _, group := range cfg.GetRuleGroups() {
			outGroups = append(outGroups, ruleGroupDTO{
				Name:        group.Name,
				Description: group.Description,
				Via:         group.Via,
				Patterns:    group.Patterns,
				PinTTL:      group.PinTTL,
			})
		}

		s.broadcast(map[string]any{"type": "rule_groups", "data": outGroups})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// handleHealth provides health check endpoint.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Minimal health payload; add more fields later if needed
	health := map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   s.version,
		"uptime":    time.Since(s.startTime).String(),
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, health)
}

func (s *Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime)

	info := serverInfoDTO{
		Version:   s.version,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		AdminPort: s.adminPort,
		DNSPort:   s.dnsPort,
		Uptime:    uptime.Round(time.Second).String(),
		BuildTime: s.buildTime,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, info)
}

// handleOverview aggregates lightweight data for the dashboard.
func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	st := s.collectStats()
	groups := s.proxy.GetRuleGroups()
	ups := s.proxy.GetConfig().Upstreams
	hist := s.proxy.History()
	// last minute aggregates
	var qLastMin, errLastMin int

	cutoff := time.Now().Add(-1 * time.Minute)
	for _, ev := range hist {
		if ev.Time.After(cutoff) {
			qLastMin++

			if ev.Status != "ok" {
				errLastMin++
			}
		}
	}

	var rulesTotal int
	for _, g := range groups {
		rulesTotal += len(g.Patterns)
	}

	payload := map[string]any{
		"stats":            st,
		"rules_total":      rulesTotal,
		"upstreams_total":  len(ups),
		"history_total":    len(hist),
		"queries_last_min": qLastMin,
		"errors_last_min":  errLastMin,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, payload)
}

// handleResolveTest runs a single DNS resolve through the active pipeline.
// GET /api/v1/resolve?name=www.youtube.com&type=A (type may be any supported name or numeric code).
func (s *Server) handleResolveTest(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		render.Status(r, defaultBadRequestStatus)
		render.JSON(w, r, map[string]string{"error": errNameRequired.Error()})

		return
	}

	qtypeStr := r.URL.Query().Get("type")

	qtype := dns.TypeA
	if qtypeStr != "" {
		qtype = parseQueryType(qtypeStr)
		if qtype == 0 {
			render.Status(r, defaultBadRequestStatus)
			render.JSON(w, r, map[string]string{"error": errUnsupportedType.Error()})

			return
		}
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(name), qtype)
	// go through proxy pipeline synchronously
	resolver := s.proxy.ResolverActive()
	if resolver == nil {
		render.Status(r, defaultInternalServerErrorStatus)
		render.JSON(w, r, map[string]string{"error": errResolverNotReady.Error()})

		return
	}

	// Measure response time
	start := time.Now()
	out, src, err := resolver.Resolve(r.Context(), m)
	responseTime := time.Since(start)

	if err != nil {
		render.Status(r, defaultBadGatewayStatus)
		render.JSON(w, r, map[string]string{"error": err.Error()})

		return
	}

	// build a compact JSON response
	resp := map[string]any{
		"upstream":         src,
		"rcode":            out.Rcode,
		"answers":          len(out.Answer),
		"records":          rrToStrings(out.Answer),
		"response_time_ms": responseTime.Milliseconds(),
	}

	// Add TTL if available
	if ttl := extractTTL(out.Answer); ttl != nil {
		resp["ttl"] = *ttl
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, resp)
}

func extractTTL(answers []dns.RR) *uint32 {
	if len(answers) == 0 {
		return nil
	}

	ttlValue := answers[0].Header().Ttl

	return &ttlValue
}

func rrToStrings(rrs []dns.RR) []string {
	out := make([]string, 0, len(rrs))
	for _, rr := range rrs {
		out = append(out, rr.String())
	}

	return out
}

func parseQueryType(qtypeStr string) uint16 {
	// Try to parse as name first (A, AAAA, MX, TXT, ...)
	if t, ok := dns.StringToType[strings.ToUpper(qtypeStr)]; ok {
		return t
	}

	// Try to parse as numeric code
	var (
		n   uint64
		err error
	)

	if strings.HasPrefix(qtypeStr, "0x") || strings.HasPrefix(qtypeStr, "0X") {
		n, err = strconv.ParseUint(qtypeStr[2:], 16, 16)
	} else {
		n, err = strconv.ParseUint(qtypeStr, 10, 16)
	}

	if err != nil {
		return 0
	}

	return uint16(n)
}

func (s *Server) buildMiddlewareChain(ctx context.Context) http.Handler {
	logger := zerolog.Ctx(ctx)

	var h http.Handler = s.mux

	// CORS
	c := cors.New(cors.Options{AllowOriginFunc: func(_ string) bool { return true }, AllowCredentials: true, AllowedHeaders: []string{"*"}})
	h = c.Handler(h)

	// Security headers
	sec := secure.New(secure.Options{
		FrameDeny:          true,
		ContentTypeNosniff: true,
		BrowserXssFilter:   true,
		ReferrerPolicy:     "strict-origin-when-cross-origin",
		ContentSecurityPolicy: "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; " +
			"script-src 'self' 'unsafe-inline'; connect-src 'self' ws: wss:",
	})
	h = sec.Handler(h)

	// Logging + request metadata
	h = hlog.NewHandler(*logger)(h)
	h = hlog.AccessHandler(func(r *http.Request, status, size int, duration time.Duration) {
		logger.Info().
			Str("method", r.Method).
			Str("url", r.URL.String()).
			Int("status", status).
			Int("size", size).
			Dur("duration", duration).
			Msg("http")
	})(h)
	h = chimw.RequestID(h)
	h = chimw.RealIP(h)
	// Recoverer last to catch panics
	h = chimw.Recoverer(h)

	// OTEL wrapper
	return otelhttp.NewHandler(h, "adminhttp")
}

func (s *Server) createServer(ctx context.Context, handler http.Handler) *http.Server {
	// Bypass middleware and otel wrappers for WebSocket upgrades to preserve http.Hijacker
	rootHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ws" {
			s.handleWS(w, r)

			return
		}

		handler.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:              s.addr,
		Handler:           rootHandler,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		IdleTimeout:       defaultIdleTimeout,
		WriteTimeout:      defaultWriteTimeout,
	}
	srv.BaseContext = func(_ net.Listener) context.Context { return ctx }

	go func() {
		<-ctx.Done()
		// graceful shutdown with timeout, then force close
		shutdownCtx, cancel := context.WithTimeout(ctx, defaultShutdownTimeout)
		defer cancel()

		srv.SetKeepAlivesEnabled(false)
		_ = srv.Shutdown(shutdownCtx)
		_ = srv.Close()
	}()

	return srv
}

// Update-related handlers

// handleUpdateCheck checks for available updates.
func (s *Server) handleUpdateCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	includePrerelease := r.URL.Query().Get("prerelease") == "true"

	updateInfo, err := s.updater.CheckForUpdates(ctx, includePrerelease)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("failed to check for updates")
		render.Status(r, defaultInternalServerErrorStatus)
		render.JSON(w, r, map[string]string{"error": err.Error()})

		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, updateInfo)
}

// handleUpdateDownload downloads the update.
func (s *Server) handleUpdateDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		DownloadURL string `json:"download_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, defaultBadRequestStatus)
		render.JSON(w, r, map[string]string{"error": err.Error()})

		return
	}

	if req.DownloadURL == "" {
		render.Status(r, defaultBadRequestStatus)
		render.JSON(w, r, map[string]string{"error": "download_url is required"})

		return
	}

	updatePath, err := s.updater.DownloadUpdate(ctx, req.DownloadURL)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("failed to download update")
		render.Status(r, defaultInternalServerErrorStatus)
		render.JSON(w, r, map[string]string{"error": err.Error()})

		return
	}

	response := map[string]string{
		"status": "downloaded",
		"path":   updatePath,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

// handleUpdateInstall installs the downloaded update.
func (s *Server) handleUpdateInstall(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		UpdatePath string `json:"update_path"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Status(r, defaultBadRequestStatus)
		render.JSON(w, r, map[string]string{"error": err.Error()})

		return
	}

	if req.UpdatePath == "" {
		render.Status(r, defaultBadRequestStatus)
		render.JSON(w, r, map[string]string{"error": "update_path is required"})

		return
	}

	err := s.updater.InstallUpdate(ctx, req.UpdatePath)
	if err != nil {
		zerolog.Ctx(ctx).Err(err).Msg("failed to install update")
		render.Status(r, defaultInternalServerErrorStatus)
		render.JSON(w, r, map[string]string{"error": err.Error()})

		return
	}

	response := map[string]string{
		"status":  "installed",
		"message": "Update installed successfully. Please restart the application.",
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

// handleUpdateStatus returns current update status.
func (s *Server) handleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(s.startTime)

	status := map[string]interface{}{
		"current_version": s.version,
		"build_time":      s.buildTime,
		"uptime":          formatDuration(uptime),
		"platform":        runtime.GOOS + "/" + runtime.GOARCH,
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, status)
}

const (
	secondsPerMinute = 60
	secondsPerHour   = 3600
)

// formatDuration formats duration to a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	}

	if d < time.Hour {
		return fmt.Sprintf("%.0fm%.0fs", d.Minutes(), d.Seconds()-d.Truncate(time.Minute).Seconds())
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) - hours*secondsPerMinute
	seconds := int(d.Seconds()) - hours*secondsPerHour - minutes*secondsPerMinute

	if hours > 0 {
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, seconds)
	}

	return fmt.Sprintf("%dm%ds", minutes, seconds)
}

// checkAndNotifyUpdates checks for updates and notifies WebSocket clients.
func (s *Server) checkAndNotifyUpdates(conn *websocket.Conn, ctx context.Context) {
	// Run in goroutine to not block WebSocket connection
	go func() {
		log := zerolog.Ctx(ctx)

		// Get update config to check if updates are enabled
		cfg := s.proxy.GetConfig()
		if !cfg.Update.Enabled {
			log.Debug().Msg("updates disabled, skipping update check")

			return
		}

		log.Debug().Msg("checking for updates on WebSocket connect")

		// Check for updates (don't include prereleases by default)
		updateInfo, err := s.updater.CheckForUpdates(ctx, cfg.Update.IncludePrerelease)
		if err != nil {
			log.Err(err).Msg("failed to check for updates on WebSocket connect")

			return
		}

		if updateInfo.HasUpdate {
			log.Info().
				Str("current_version", updateInfo.CurrentVersion).
				Str("latest_version", updateInfo.LatestVersion).
				Msg("update available, notifying WebSocket client")

			// Send update notification to this specific WebSocket connection
			s.sendJSON(conn, map[string]any{
				"type": "update_available",
				"data": map[string]any{
					"current_version": updateInfo.CurrentVersion,
					"latest_version":  updateInfo.LatestVersion,
					"release": map[string]any{
						"name":         updateInfo.Release.Name,
						"body":         updateInfo.Release.Body,
						"published_at": updateInfo.Release.PublishedAt,
						"assets":       updateInfo.Release.Assets,
					},
				},
			})
		} else {
			log.Debug().Msg("no updates available")
		}
	}()
}
