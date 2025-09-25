package dnsproxy

import (
	"context"
	"strconv"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/miekg/dns"
	"golang.org/x/sync/singleflight"
)

type cacheItem struct {
	msg    *dns.Msg
	expire time.Time
}

type CachedResolver struct {
	Next       Resolver
	MaxEntries int

	lru *lru.LRU[string, cacheItem]
	sf  singleflight.Group
}

func NewCachedResolver(next Resolver, maxEntries int) *CachedResolver {
	if maxEntries <= 0 {
		maxEntries = 10000
	}
	// default expiration 0 (we handle per-entry TTL manually)
	l := lru.NewLRU[string, cacheItem](maxEntries, nil, 0)

	return &CachedResolver{Next: next, MaxEntries: maxEntries, lru: l}
}

func (c *CachedResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if q == nil || len(q.Question) == 0 {
		return c.Next.Resolve(ctx, q)
	}

	key := strings.ToLower(strings.TrimSuffix(q.Question[0].Name, ".")) + ":" + strconv.FormatUint(uint64(q.Question[0].Qtype), 10)

	it, ok := c.lru.Get(key)
	if !ok {
		// Coalesce concurrent cache misses for the same key
		v, err, _ := c.sf.Do(key, func() (any, error) {
			out, src, err := c.resolveAndCache(ctx, q, key)
			if err != nil || out == nil {
				return nil, err
			}

			return struct {
				msg *dns.Msg
				src string
			}{msg: out, src: src}, nil
		})
		if err != nil || v == nil {
			return nil, "", err
		}

		res, ok2 := v.(struct {
			msg *dns.Msg
			src string
		})
		if !ok2 {
			return nil, "", nil
		}

		return res.msg, res.src, nil
	}

	if time.Now().After(it.expire) {
		c.lru.Remove(key)

		return c.resolveAndCache(ctx, q, key)
	}

	// Build reply with minimal allocations: reuse cached slices (immutable)
	reply := new(dns.Msg)
	reply.SetReply(q)
	reply.RecursionAvailable = it.msg.RecursionAvailable
	reply.Authoritative = it.msg.Authoritative
	reply.Rcode = it.msg.Rcode
	reply.Answer = it.msg.Answer
	reply.Ns = it.msg.Ns
	reply.Extra = it.msg.Extra

	return reply, "cache", nil
}

func (c *CachedResolver) resolveAndCache(ctx context.Context, q *dns.Msg, key string) (*dns.Msg, string, error) {
	out, src, err := c.Next.Resolve(ctx, q)
	if err == nil && out != nil {
		c.put(key, out)
	}

	return out, src, err
}

func (c *CachedResolver) put(key string, msg *dns.Msg) {
	if msg == nil {
		return
	}

	ttl := ttlFromMsg(msg)
	if ttl <= 0 {
		ttl = 30
	}
	// store copy with per-entry expiration
	it := cacheItem{msg: msg.Copy(), expire: time.Now().Add(time.Duration(ttl) * time.Second)}
	c.lru.Add(key, it)
	// metrics moved to MetricsResolver; cache exposes only storage behavior
}

func ttlFromMsg(msg *dns.Msg) uint32 {
	var minTTL uint32

	for _, rr := range msg.Answer {
		if rr == nil {
			continue
		}

		if h := rr.Header(); h != nil {
			if minTTL == 0 || h.Ttl < minTTL {
				minTTL = h.Ttl
			}
		}
	}

	return minTTL
}
