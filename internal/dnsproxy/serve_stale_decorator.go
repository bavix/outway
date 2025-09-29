package dnsproxy

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

type ServeStaleResolver struct {
	Cache *CachedResolver
}

const sourceCache = "cache"

func (s *ServeStaleResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if q == nil || len(q.Question) == 0 || s.Cache == nil {
		return s.Cache.Next.Resolve(ctx, q)
	}

	key := strings.ToLower(strings.TrimSuffix(q.Question[0].Name, ".")) + ":" + strconv.FormatUint(uint64(q.Question[0].Qtype), 10)

	// Try to read from cache, even if expired
	if it, ok := s.Cache.lru.Get(key); ok {
		// If not expired, let cache resolver handle it normally for metrics and reuse
		if !nowAfter(it.expire) {
			return s.Cache.Resolve(ctx, q)
		}

		// Serve stale immediately and refresh in background using singleflight
		go func() {
			_, _, _ = s.Cache.sf.Do(key, func() (any, error) {
				_, _, err := s.Cache.resolveAndCache(ctx, q, key)

				return nil, err
			})
		}()

		reply := new(dns.Msg)
		reply.SetReply(q)
		reply.RecursionAvailable = it.msg.RecursionAvailable
		reply.Authoritative = it.msg.Authoritative
		reply.Rcode = it.msg.Rcode
		reply.Answer = it.msg.Answer
		reply.Ns = it.msg.Ns
		reply.Extra = it.msg.Extra

		return reply, sourceCache, nil
	}

	// Miss: fall through to cache resolver which will populate on success
	return s.Cache.Resolve(ctx, q)
}

// nowAfter is a tiny wrapper to allow testing/injection if needed later.
func nowAfter(t time.Time) bool { return time.Now().After(t) }
