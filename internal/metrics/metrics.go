//nolint:gochecknoglobals // prometheus metrics and global state
package metrics

import (
	"errors"
	"strconv"
	"sync/atomic"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"
)

const (
	msToSecondsDivisor = 1000.0
)

var (
	DNSQueriesTotal = promauto.NewCounterVec(
		prom.CounterOpts{
			Name: "dns_client_queries_total",
			Help: "Total DNS queries processed by the proxy (Counter).",
		},
		[]string{"service"},
	)
	DNSMarksTotal = promauto.NewCounterVec(
		prom.CounterOpts{
			Name: "dns_mark_operations_total",
			Help: "IP mark operations by outcome (Counter). outcome=success|error|dropped.",
		},
		[]string{"service", "outcome"},
	)
	AdminRequestsTotal = promauto.NewCounterVec(
		prom.CounterOpts{
			Name: "http_server_requests_total",
			Help: "Admin HTTP requests handled (Counter). Labels: service, method, route, status.",
		},
		[]string{"service", "method", "route", "status"},
	)

	MarksDroppedTotal = promauto.NewCounterVec(
		prom.CounterOpts{
			Name: "dns_marks_dropped_total",
			Help: "Total dropped mark operations (e.g., quota exceeded)",
		},
		[]string{"service"},
	)

	CurrentTrackedIPsPerIface = promauto.NewGaugeVec(
		prom.GaugeOpts{
			Name: "egress_tracked_ips_per_interface",
			Help: "Currently tracked IPs per interface (Gauge).",
		},
		[]string{"service", "iface"},
	)
	CurrentTrackedIPsTotal = promauto.NewGaugeVec(
		prom.GaugeOpts{
			Name: "egress_tracked_ips",
			Help: "Current total number of tracked IPs (Gauge).",
		},
		[]string{"service"},
	)
	ReadyGauge = promauto.NewGaugeVec(
		prom.GaugeOpts{
			Name: "service_ready",
			Help: "Service readiness: 1=ready, 0=not ready (Gauge).",
		},
		[]string{"service"},
	)

	DNSUpstreamRTT = promauto.NewHistogramVec(prom.HistogramOpts{
		Name:    "dns_upstream_rtt_seconds",
		Help:    "Upstream DNS RTT in seconds (Histogram).",
		Buckets: []float64{0.001, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1.0},
	}, []string{"service"})
	DNSRequestDuration = promauto.NewHistogramVec(prom.HistogramOpts{
		Name:    "dns_request_duration_seconds",
		Help:    "End-to-end DNS request duration in seconds (Histogram).",
		Buckets: []float64{0.0001, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1.0},
	}, []string{"service"})

	// DNSRequestDurationByUpstream is labeled per upstream.
	DNSRequestDurationByUpstream = promauto.NewHistogramVec(prom.HistogramOpts{
		Name:    "dns_request_duration_seconds_by_upstream",
		Help:    "DNS request duration in seconds by upstream (Histogram).",
		Buckets: []float64{0.0001, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1.0},
	}, []string{"service", "upstream"})
	ResolveErrorsTotal = promauto.NewCounterVec(prom.CounterOpts{
		Name: "dns_resolve_errors_total",
		Help: "Total resolve errors by upstream (Counter).",
	}, []string{"service", "upstream"})

	// CacheHitsTotal contains cache metrics.
	CacheHitsTotal = promauto.NewCounterVec(
		prom.CounterOpts{
			Name: "dns_cache_hits_total",
			Help: "Total cache hits.",
		},
		[]string{"service"},
	)
	CacheMissesTotal = promauto.NewCounterVec(
		prom.CounterOpts{
			Name: "dns_cache_misses_total",
			Help: "Total cache misses.",
		},
		[]string{"service"},
	)
	CacheEvictionsTotal = promauto.NewCounterVec(
		prom.CounterOpts{
			Name: "dns_cache_evictions_total",
			Help: "Total cache evictions.",
		},
		[]string{"service"},
	)
	CacheEntries = promauto.NewGaugeVec(
		prom.GaugeOpts{
			Name: "dns_cache_entries",
			Help: "Current number of cache entries.",
		},
		[]string{"service"},
	)
	CacheBytes = promauto.NewGaugeVec(
		prom.GaugeOpts{
			Name: "dns_cache_bytes",
			Help: "Approximate cache size in bytes.",
		},
		[]string{"service"},
	)
)

var readyFlag int32 //nolint:gochecknoglobals // service ready flag

var serviceName atomic.Value //nolint:gochecknoglobals // service name // string

// SetService sets the service label value (default: outway).
func SetService(name string) { serviceName.Store(name) }

func Service() string {
	if v := serviceName.Load(); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}

	return "outway"
}

// RegisterCollectors registers default Go and process collectors.
// Should be called once during program startup (e.g., in cmd).
func RegisterCollectors() {
	registerDefault(collectors.NewGoCollector())
	registerDefault(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
}

func registerDefault(c prom.Collector) {
	if err := prom.Register(c); err != nil {
		var are prom.AlreadyRegisteredError
		if errors.As(err, &are) {
			return
		}
		// best-effort: ignore unexpected errors to avoid panics in init
	}
}

var M struct { //nolint:gochecknoglobals // metrics cache
	DNSQueries              prom.Counter
	DNSMarksSuccess         prom.Counter
	DNSMarksError           prom.Counter
	DNSMarksDropped         prom.Counter
	UpstreamRTT             prom.Observer
	RequestDuration         prom.Observer
	RequestDurationUpstream *prom.HistogramVec
	ResolveErrorsByUpstream *prom.CounterVec

	CacheHits      prom.Counter
	CacheMisses    prom.Counter
	CacheEvictions prom.Counter
	CacheEntries   prom.Gauge
	CacheBytes     prom.Gauge
}

func BindService() {
	s := Service()
	M.DNSQueries = DNSQueriesTotal.WithLabelValues(s)
	M.DNSMarksSuccess = DNSMarksTotal.WithLabelValues(s, "success")
	M.DNSMarksError = DNSMarksTotal.WithLabelValues(s, "error")
	M.DNSMarksDropped = DNSMarksTotal.WithLabelValues(s, "dropped")
	M.UpstreamRTT = DNSUpstreamRTT.WithLabelValues(s)
	M.RequestDuration = DNSRequestDuration.WithLabelValues(s)
	M.RequestDurationUpstream = DNSRequestDurationByUpstream
	M.ResolveErrorsByUpstream = ResolveErrorsTotal

	// Cache bindings
	M.CacheHits = CacheHitsTotal.WithLabelValues(s)
	M.CacheMisses = CacheMissesTotal.WithLabelValues(s)
	M.CacheEvictions = CacheEvictionsTotal.WithLabelValues(s)
	M.CacheEntries = CacheEntries.WithLabelValues(s)
	M.CacheBytes = CacheBytes.WithLabelValues(s)
}

// ObserveRequestDurationUpstream records duration with upstream label.
func ObserveRequestDurationUpstream(upstream string, sec float64) {
	if upstream == "" {
		upstream = "unknown"
	}

	DNSRequestDurationByUpstream.WithLabelValues(Service(), upstream).Observe(sec)
}

// IncResolveError increments error counter for upstream.
func IncResolveError(upstream string) {
	if upstream == "" {
		upstream = "unknown"
	}

	ResolveErrorsTotal.WithLabelValues(Service(), upstream).Inc()
}

// Simple in-memory RPS ring (per process).
const rpsWindow = 60

var (
	rpsHits    [rpsWindow]uint64
	rpsMisses  [rpsWindow]uint64
	rpsIndex   int64 // atomic
	rpsTickSet int32
)

// StartRPSTicker starts a background ticker that advances the ring each second.
func StartRPSTicker() {
	if !atomic.CompareAndSwapInt32(&rpsTickSet, 0, 1) {
		return
	}

	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()

		for range t.C {
			i := int(atomic.AddInt64(&rpsIndex, 1) % rpsWindow)
			atomic.StoreUint64(&rpsHits[i], 0)
			atomic.StoreUint64(&rpsMisses[i], 0)
		}
	}()
}

// RecordCacheHit increments current second bucket for hits.
func RecordCacheHit() {
	i := int(atomic.LoadInt64(&rpsIndex) % rpsWindow)
	atomic.AddUint64(&rpsHits[i], 1)
}

// RecordCacheMiss increments current second bucket for misses.
func RecordCacheMiss() {
	i := int(atomic.LoadInt64(&rpsIndex) % rpsWindow)
	atomic.AddUint64(&rpsMisses[i], 1)
}

func snapshotRPS() (uint64, uint64) {
	var hits, misses uint64
	for i := range rpsWindow {
		hits += atomic.LoadUint64(&rpsHits[i])
		misses += atomic.LoadUint64(&rpsMisses[i])
	}

	return hits, misses
}

// RecordHTTP increments admin HTTP requests with OTEL-style labels.
func RecordHTTP(method, route string, status int) {
	AdminRequestsTotal.WithLabelValues(Service(), method, route, strconv.Itoa(status)).Inc()
}

// metrics server is provided by adminhttp; no separate server here

// SetReady sets readiness and updates the gauge.
func SetReady(v bool) {
	if v {
		atomic.StoreInt32(&readyFlag, 1)
		ReadyGauge.WithLabelValues(Service()).Set(1)
	} else {
		atomic.StoreInt32(&readyFlag, 0)
		ReadyGauge.WithLabelValues(Service()).Set(0)
	}
}

// IsReady returns current readiness flag.
func IsReady() bool { return atomic.LoadInt32(&readyFlag) == 1 }

// Stats represents a lightweight analytics snapshot for the admin UI.
type Stats struct {
	DNSQueriesTotal          float64 `json:"dns_queries_total"`
	DNSMarksTotal            float64 `json:"dns_marks_total"`
	MarksDroppedTotal        float64 `json:"marks_dropped_total"`
	DNSUpstreamRTTAvgSeconds float64 `json:"dns_upstream_rtt_avg_seconds"`
	DNSRequestAvgSeconds     float64 `json:"dns_request_avg_seconds"`
	ServiceReady             float64 `json:"service_ready"`
	EffectiveRPS             float64 `json:"effective_rps"`
	OriginRPS                float64 `json:"origin_rps"`
	CacheHitRate             float64 `json:"cache_hit_rate"`
}

// GatherStats collects basic stats from the default registry for a given service label.
//
//nolint:gocyclo // Complex metric gathering logic with many conditional branches
func GatherStats(service string) (Stats, error) { //nolint:gocognit,cyclop,funlen
	mfs, err := prom.DefaultGatherer.Gather()
	if err != nil {
		return Stats{}, err
	}

	var (
		s                                  Stats
		rttSum, rttCount, reqSum, reqCount float64
	)

	withService := func(m *dto.Metric) bool {
		for _, lp := range m.GetLabel() {
			if lp.GetName() == "service" && lp.GetValue() == service {
				return true
			}
		}

		return false
	}

	for _, mf := range mfs {
		name := mf.GetName()
		switch name {
		case "dns_client_queries_total":
			for _, m := range mf.GetMetric() {
				if withService(m) {
					s.DNSQueriesTotal += m.GetCounter().GetValue()
				}
			}
		case "dns_mark_operations_total":
			for _, m := range mf.GetMetric() {
				if withService(m) {
					s.DNSMarksTotal += m.GetCounter().GetValue()
				}
			}
		case "dns_marks_dropped_total":
			for _, m := range mf.GetMetric() {
				if withService(m) {
					s.MarksDroppedTotal += m.GetCounter().GetValue()
				}
			}
		case "dns_upstream_rtt_seconds":
			for _, m := range mf.GetMetric() {
				if withService(m) {
					h := m.GetHistogram()
					rttSum += h.GetSampleSum() / msToSecondsDivisor
					rttCount += float64(h.GetSampleCount())
				}
			}
		case "dns_request_duration_seconds":
			for _, m := range mf.GetMetric() {
				if withService(m) {
					h := m.GetHistogram()
					reqSum += h.GetSampleSum() / msToSecondsDivisor
					reqCount += float64(h.GetSampleCount())
				}
			}
		case "service_ready":
			for _, m := range mf.GetMetric() {
				if withService(m) {
					s.ServiceReady = m.GetGauge().GetValue()
				}
			}
		}
	}

	if rttCount > 0 {
		s.DNSUpstreamRTTAvgSeconds = rttSum / rttCount
	}

	if reqCount > 0 {
		s.DNSRequestAvgSeconds = reqSum / reqCount
	}

	// derive RPS metrics from ring
	hits, misses := snapshotRPS()
	eff := float64(hits+misses) / float64(rpsWindow)
	orig := float64(misses) / float64(rpsWindow)
	s.EffectiveRPS = eff
	s.OriginRPS = orig

	// Calculate cache hit rate from Prometheus metrics (total history)
	var totalCacheHits, totalCacheMisses float64

	for _, mf := range mfs {
		name := mf.GetName()
		switch name {
		case "dns_cache_hits_total":
			for _, m := range mf.GetMetric() {
				if withService(m) {
					totalCacheHits += m.GetCounter().GetValue()
				}
			}
		case "dns_cache_misses_total":
			for _, m := range mf.GetMetric() {
				if withService(m) {
					totalCacheMisses += m.GetCounter().GetValue()
				}
			}
		}
	}

	// Use Prometheus metrics for cache hit rate (total history)
	if totalCacheHits+totalCacheMisses > 0 {
		s.CacheHitRate = totalCacheHits / (totalCacheHits + totalCacheMisses)
	}

	return s, nil
}
