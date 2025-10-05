package dnsproxy

import (
	"context"
	"sort"
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

// cacheChangeNotify is an optional hook set by dashboardhttp to broadcast cache changes.
//
//nolint:gochecknoglobals
var cacheChangeNotify func()

// SetCacheChangeNotifier sets a global hook invoked on cache mutations/evictions.
func SetCacheChangeNotifier(fn func()) { cacheChangeNotify = fn }

type CachedResolver struct {
	Next          Resolver
	MaxEntries    int
	MinTTLSeconds int
	MaxTTLSeconds int

	lru *lru.LRU[string, cacheItem]
	sf  singleflight.Group
}

func NewCachedResolver(next Resolver, maxEntries int, minTTLSeconds int, maxTTLSeconds int) *CachedResolver {
	l := lru.NewLRU[string, cacheItem](maxEntries, nil, 0)

	return &CachedResolver{
		Next:          next,
		MaxEntries:    maxEntries,
		MinTTLSeconds: minTTLSeconds,
		MaxTTLSeconds: maxTTLSeconds,
		lru:           l,
	}
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

//nolint:funcorder // keep helper close to Resolve for readability
func (c *CachedResolver) resolveAndCache(ctx context.Context, q *dns.Msg, key string) (*dns.Msg, string, error) {
	out, src, err := c.Next.Resolve(ctx, q)
	// Skip caching when there are no answers (empty result pool)
	if err == nil && out != nil && len(out.Answer) > 0 {
		c.put(key, out)
	}

	return out, src, err
}

//nolint:funcorder // helper grouped with resolveAndCache
func (c *CachedResolver) put(key string, msg *dns.Msg) {
	if msg == nil {
		return
	}

	ttl := ttlFromMsg(msg)
	if ttl <= 0 {
		ttl = uint32(c.MinTTLSeconds) //nolint:gosec // TTL bounds validated in config
	}

	t := max(c.MinTTLSeconds, min(int(ttl), c.MaxTTLSeconds))
	ttl = uint32(t) //nolint:gosec // TTL bounds validated in config
	it := cacheItem{msg: msg.Copy(), expire: time.Now().Add(time.Duration(ttl) * time.Second)}
	c.lru.Add(key, it)

	if cacheChangeNotify != nil {
		cacheChangeNotify()
	}
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

// Flush clears all cache entries.
func (c *CachedResolver) Flush() {
	if c == nil || c.lru == nil {
		return
	}

	c.lru.Purge()

	if cacheChangeNotify != nil {
		cacheChangeNotify()
	}
}

// Delete removes cache entries for a specific name and qtype.
// If qtype is 0, deletes common types for the name.
func (c *CachedResolver) Delete(name string, qtype uint16) {
	if c == nil || c.lru == nil {
		return
	}

	name = strings.ToLower(strings.TrimSuffix(name, "."))
	if name == "" {
		return
	}

	if qtype != 0 {
		key := name + ":" + strconv.FormatUint(uint64(qtype), 10)
		c.lru.Remove(key)

		if cacheChangeNotify != nil {
			cacheChangeNotify()
		}

		return
	}
	// Remove a set of common qtypes for this name
	common := []uint16{
		dns.TypeA, dns.TypeAAAA, dns.TypeCNAME, dns.TypeMX, dns.TypeNS,
		dns.TypeTXT, dns.TypeSRV, dns.TypePTR,
	}
	for _, t := range common {
		key := name + ":" + strconv.FormatUint(uint64(t), 10)
		c.lru.Remove(key)

		if cacheChangeNotify != nil {
			cacheChangeNotify()
		}
	}
}

// cacheEntry is a compact DTO for admin listing.
type cacheEntry struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"`
	QType     uint16    `json:"qtype"`
	Answers   int       `json:"answers"`
	ExpiresAt time.Time `json:"expires_at"`
	RCode     int       `json:"rcode"`
}

// List returns a paginated list of non-expired cache entries filtered by substring q.
// sortBy: name|expires|qtype|answers, order: asc|desc.
//
//nolint:gocognit,cyclop,funlen,nonamedreturns // sorting and pagination branching kept explicit for clarity
func (c *CachedResolver) List(offset, limit int, q string, sortBy, order string) (items []cacheEntry, total int) {
	if c == nil {
		return nil, 0
	}

	// Build snapshot from underlying LRU keys
	keys := c.lru.Keys()
	tmp := make([]cacheEntry, 0, len(keys))
	now := time.Now()
	ql := strings.ToLower(q)

	for _, k := range keys {
		it, ok := c.lru.Get(k)
		if !ok || now.After(it.expire) {
			continue
		}

		const keyParts = 2

		parts := strings.SplitN(k, ":", keyParts)
		if len(parts) != keyParts {
			continue
		}

		name := parts[0]
		if ql != "" && !strings.Contains(name, ql) {
			continue
		}

		qtype64, _ := strconv.ParseUint(parts[1], 10, 16)
		tmp = append(tmp, cacheEntry{
			Key:       k,
			Name:      name,
			QType:     uint16(qtype64),
			Answers:   len(it.msg.Answer),
			ExpiresAt: it.expire,
			RCode:     it.msg.Rcode,
		})
	}

	total = len(tmp)
	sb := strings.ToLower(sortBy)
	so := strings.ToLower(order)
	desc := so == "desc" || so == "" // default desc by expires

	sort.Slice(tmp, func(i, j int) bool {
		switch sb {
		case "name":
			if desc {
				if tmp[i].Name == tmp[j].Name {
					return tmp[i].QType > tmp[j].QType
				}

				return tmp[i].Name > tmp[j].Name
			}

			if tmp[i].Name == tmp[j].Name {
				return tmp[i].QType < tmp[j].QType
			}

			return tmp[i].Name < tmp[j].Name
		case "qtype":
			if desc {
				return tmp[i].QType > tmp[j].QType
			}

			return tmp[i].QType < tmp[j].QType
		case "answers":
			if desc {
				return tmp[i].Answers > tmp[j].Answers
			}

			return tmp[i].Answers < tmp[j].Answers
		default: // expires
			if desc {
				return tmp[i].ExpiresAt.After(tmp[j].ExpiresAt)
			}

			return tmp[i].ExpiresAt.Before(tmp[j].ExpiresAt)
		}
	})

	if offset < 0 {
		offset = 0
	}

	if limit <= 0 {
		limit = 50
	}

	if offset > len(tmp) {
		return []cacheEntry{}, total
	}

	end := offset + limit
	if end > len(tmp) {
		end = len(tmp)
	}

	items = tmp[offset:end]

	return items, total
}

// Get returns detailed information for a specific key (if not expired).
func (c *CachedResolver) Get(key string) (*dns.Msg, bool) {
	if c == nil || c.lru == nil {
		return nil, false
	}

	if it, ok := c.lru.Get(key); ok && !time.Now().After(it.expire) {
		return it.msg.Copy(), true
	}

	return nil, false
}

// DeleteKey removes a specific entry by exact key.
func (c *CachedResolver) DeleteKey(key string) {
	if c == nil || c.lru == nil {
		return
	}

	c.lru.Remove(key)
}
