package dnsproxy

import (
	"context"
	"time"

	"github.com/miekg/dns"

	"github.com/bavix/outway/internal/metrics"
)

type MetricsResolver struct{ Next Resolver }

func (m *MetricsResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	start := time.Now()

	metrics.M.DNSQueries.Inc()

	out, src, err := m.Next.Resolve(ctx, q)
	durSec := time.Since(start).Seconds()
	metrics.M.RequestDuration.Observe(durSec)
	metrics.ObserveRequestDurationUpstream(src, durSec)

	if err != nil {
		metrics.IncResolveError(src)
	}

	// Cache hit/miss accounting at a single place
	if src == "cache" {
		if metrics.M.CacheHits != nil {
			metrics.M.CacheHits.Inc()
		}

		metrics.RecordCacheHit()
	} else {
		if metrics.M.CacheMisses != nil {
			metrics.M.CacheMisses.Inc()
		}

		metrics.RecordCacheMiss()
	}

	return out, src, err
}
