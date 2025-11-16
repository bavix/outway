package dnsproxy

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/firewall"
	"github.com/bavix/outway/internal/metrics"
)

// markRequest represents a request to mark an IP address.
type markRequest struct {
	ip        string
	iface     string
	ttl       int
	timestamp time.Time
}

// AsyncMarkResolver performs IP marking asynchronously with debounce and caching.
type AsyncMarkResolver struct {
	Next    Resolver
	Backend firewall.Backend
	Rules   *RuleStore
	Cfg     *config.Config

	// Async marking state
	mu            sync.RWMutex
	pendingMarks  map[string]*markRequest // "ip:iface" -> request
	markedIPs     map[string]time.Time    // "ip:iface" -> expiry time
	debounceTimer *time.Timer
	debounceDelay time.Duration
	workerRunning bool
	workerStop    chan struct{}
	workerWg      sync.WaitGroup
}

const (
	// Default debounce delay for batching mark requests.
	defaultDebounceDelay = 100 * time.Millisecond
	// Cache expiry buffer - mark IPs slightly before they expire.
	cacheExpiryBuffer = 5 * time.Second
)

// NewAsyncMarkResolver creates a new async mark resolver.
func NewAsyncMarkResolver(next Resolver, backend firewall.Backend, rules *RuleStore, cfg *config.Config) *AsyncMarkResolver {
	resolver := &AsyncMarkResolver{
		Next:          next,
		Backend:       backend,
		Rules:         rules,
		Cfg:           cfg,
		pendingMarks:  make(map[string]*markRequest),
		markedIPs:     make(map[string]time.Time),
		debounceDelay: defaultDebounceDelay,
		workerStop:    make(chan struct{}),
	}

	// Start background worker
	resolver.startWorker()

	return resolver
}

// Resolve resolves DNS query and queues IP marking asynchronously.
func (m *AsyncMarkResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	out, src, err := m.Next.Resolve(ctx, q)
	if err != nil || out == nil || len(out.Answer) == 0 || m.Backend == nil || m.Rules == nil || q == nil || len(q.Question) == 0 {
		return out, src, err
	}

	name := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(q.Question[0].Name, ".")))

	rule, ok := m.Rules.Find(name)
	if !ok {
		return out, src, err
	}

	// Queue IPs for async marking (non-blocking)
	m.queueMarks(ctx, out.Answer, rule, name)

	return out, src, err
}

// Stop gracefully stops the AsyncMarkResolver.
func (m *AsyncMarkResolver) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.workerRunning {
		return
	}

	m.workerRunning = false
	close(m.workerStop)
	m.workerWg.Wait()

	// Process any remaining marks before stopping
	if m.debounceTimer != nil {
		m.debounceTimer.Stop()
	}

	// Use background context for cleanup operation
	m.processPendingMarks(context.Background())
}

// queueMarks queues IP addresses for async marking.
//
//nolint:funlen // complex IP extraction and queuing logic
func (m *AsyncMarkResolver) queueMarks(ctx context.Context, answers []dns.RR, rule config.Rule, domain string) {
	now := time.Now()

	for _, rr := range answers {
		var (
			ip  string
			ttl uint32
		)

		switch a := rr.(type) {
		case *dns.A:
			ip = a.A.String()

			ttl = a.Hdr.Ttl
			if rule.PinTTL {
				ttl = uint32(m.Cfg.GetMinMarkTTL(ttl).Seconds())
			} else {
				ttl = minTTL(ttl)
			}
		case *dns.AAAA:
			ip = a.AAAA.String()

			ttl = a.Hdr.Ttl
			if rule.PinTTL {
				ttl = uint32(m.Cfg.GetMinMarkTTL(ttl).Seconds())
			} else {
				ttl = minTTL(ttl)
			}
		default:
			continue
		}

		// Check cache first - skip if already marked and not expired
		cacheKey := ip + ":" + rule.Via

		m.mu.RLock()

		if expiry, exists := m.markedIPs[cacheKey]; exists && now.Before(expiry.Add(-cacheExpiryBuffer)) {
			m.mu.RUnlock()
			zerolog.Ctx(ctx).Debug().
				Str("domain", domain).
				Str("ip", ip).
				Str("via", rule.Via).
				Msg("IP already marked (cached), skipping")

			continue
		}

		m.mu.RUnlock()

		// Queue for async marking
		m.mu.Lock()
		m.pendingMarks[cacheKey] = &markRequest{
			ip:        ip,
			iface:     rule.Via,
			ttl:       int(ttl),
			timestamp: now,
		}

		// Reset debounce timer
		if m.debounceTimer != nil {
			m.debounceTimer.Stop()
		}

		// Capture context for debounce callback
		debounceCtx := ctx
		m.debounceTimer = time.AfterFunc(m.debounceDelay, func() {
			m.processPendingMarks(debounceCtx)
		})
		m.mu.Unlock()

		zerolog.Ctx(ctx).Debug().
			Str("domain", domain).
			Str("ip", ip).
			Str("via", rule.Via).
			Int("ttl", int(ttl)).
			Msg("IP queued for async marking")
	}
}

// processPendingMarks processes all pending mark requests in batch.
func (m *AsyncMarkResolver) processPendingMarks(ctx context.Context) {
	m.mu.Lock()

	if len(m.pendingMarks) == 0 {
		m.mu.Unlock()

		return
	}

	// Copy pending marks and clear
	marks := make([]*markRequest, 0, len(m.pendingMarks))
	for _, req := range m.pendingMarks {
		marks = append(marks, req)
	}

	m.pendingMarks = make(map[string]*markRequest)
	m.mu.Unlock()

	// Process marks in background
	go m.markIPsBatch(ctx, marks)
}

// markIPsBatch marks a batch of IP addresses.
func (m *AsyncMarkResolver) markIPsBatch(ctx context.Context, marks []*markRequest) {
	logger := zerolog.Ctx(ctx).With().Int("batch_size", len(marks)).Logger()

	logger.Debug().Msg("processing batch of IP marks")

	now := time.Now()
	successCount := 0
	errorCount := 0

	for _, req := range marks {
		cacheKey := req.ip + ":" + req.iface

		// Double-check cache (another goroutine might have marked it)
		m.mu.RLock()

		if expiry, exists := m.markedIPs[cacheKey]; exists && now.Before(expiry.Add(-cacheExpiryBuffer)) {
			m.mu.RUnlock()
			logger.Debug().
				Str("ip", req.ip).
				Str("iface", req.iface).
				Msg("IP already marked (skipping duplicate)")

			continue
		}

		m.mu.RUnlock()

		// Mark IP using batch context
		if err := m.Backend.MarkIP(ctx, req.iface, req.ip, req.ttl); err != nil {
			logger.Error().
				Err(err).
				Str("ip", req.ip).
				Str("iface", req.iface).
				Int("ttl", req.ttl).
				Msg("failed to mark IP")
			metrics.M.DNSMarksError.Inc()

			errorCount++
		} else {
			// Update cache
			expiry := now.Add(time.Duration(req.ttl) * time.Second)

			m.mu.Lock()
			m.markedIPs[cacheKey] = expiry
			m.mu.Unlock()

			logger.Debug().
				Str("ip", req.ip).
				Str("iface", req.iface).
				Int("ttl", req.ttl).
				Msg("IP marked successfully")
			metrics.M.DNSMarksSuccess.Inc()

			successCount++
		}
	}

	logger.Info().
		Int("success", successCount).
		Int("errors", errorCount).
		Msg("batch IP marking completed")
}

// startWorker starts background worker for cache cleanup.
func (m *AsyncMarkResolver) startWorker() {
	m.mu.Lock()

	if m.workerRunning {
		m.mu.Unlock()

		return
	}

	m.workerRunning = true
	m.mu.Unlock()

	m.workerWg.Go(func() {
		const cacheCleanupInterval = 30 * time.Second

		ticker := time.NewTicker(cacheCleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				m.cleanupCache()
			case <-m.workerStop:
				return
			}
		}
	})
}

// cleanupCache removes expired entries from cache.
func (m *AsyncMarkResolver) cleanupCache() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, expiry := range m.markedIPs {
		if now.After(expiry) {
			delete(m.markedIPs, key)

			removed++
		}
	}

	if removed > 0 {
		zerolog.Ctx(context.Background()).Debug().
			Int("removed", removed).
			Int("remaining", len(m.markedIPs)).
			Msg("cleaned up expired IP marks from cache")
	}
}
