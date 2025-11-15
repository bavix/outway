package dnsproxy

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/miekg/dns"
	"github.com/rs/zerolog"
	"golang.org/x/sync/singleflight"
)

const (
	bytesPerKB = 1024
	bytesPerMB = bytesPerKB * bytesPerKB
	// Approximate sizes for DNS message components.
	dnsHeaderSize       = 12
	dnsQuestionOverhead = 4  // qtype + qclass
	avgDNSRecordSize    = 35 // Average size of a DNS record
)

type cacheItem struct {
	msg    *dns.Msg
	expire time.Time
	size   int64 // Approximate size in bytes
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
	MaxSizeBytes  int64 // Maximum cache size in bytes (0 = disabled)
	MinTTLSeconds int
	MaxTTLSeconds int

	lru         *lru.LRU[string, cacheItem]
	sf          singleflight.Group
	currentSize int64 // Current cache size in bytes
	sizeMu      sync.RWMutex
}

func NewCachedResolver(next Resolver, maxEntries int, minTTLSeconds int, maxTTLSeconds int) *CachedResolver {
	return NewCachedResolverWithSize(next, maxEntries, 0, minTTLSeconds, maxTTLSeconds)
}

// NewCachedResolverWithSize creates a new cached resolver with size limit.
// This is exported for testing purposes.
func NewCachedResolverWithSize(next Resolver, maxEntries int, maxSizeMB int, minTTLSeconds int, maxTTLSeconds int) *CachedResolver {
	l := lru.NewLRU[string, cacheItem](maxEntries, nil, 0)

	var maxSizeBytes int64
	if maxSizeMB > 0 {
		maxSizeBytes = int64(maxSizeMB) * bytesPerMB // Convert MB to bytes
	}

	return &CachedResolver{
		Next:          next,
		MaxEntries:    maxEntries,
		MaxSizeBytes:  maxSizeBytes,
		MinTTLSeconds: minTTLSeconds,
		MaxTTLSeconds: maxTTLSeconds,
		lru:           l,
		currentSize:   0,
	}
}

//nolint:cyclop,funlen // complex branching for cache hit/miss/expiry logic
func (c *CachedResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if q == nil || len(q.Question) == 0 {
		return c.Next.Resolve(ctx, q)
	}

	key := strings.ToLower(strings.TrimSuffix(q.Question[0].Name, ".")) + ":" + strconv.FormatUint(uint64(q.Question[0].Qtype), 10)

	it, ok := c.lru.Get(key)
	if !ok {
		// Coalesce concurrent cache misses for the same key
		zerolog.Ctx(ctx).Debug().
			Str("cache_key", key).
			Str("query", strings.TrimSuffix(q.Question[0].Name, ".")).
			Uint16("qtype", q.Question[0].Qtype).
			Msg("cache miss")

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
		// Update size tracking when removing expired entry
		if c.MaxSizeBytes > 0 {
			c.sizeMu.Lock()
			c.currentSize -= it.size
			c.sizeMu.Unlock()
		}

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
		c.put(ctx, key, out)
	}

	return out, src, err
}

//nolint:funcorder,cyclop,nestif,funlen // helper grouped with resolveAndCache, complex eviction logic
func (c *CachedResolver) put(ctx context.Context, key string, msg *dns.Msg) {
	if msg == nil {
		return
	}

	ttl := ttlFromMsg(msg)
	if ttl <= 0 {
		ttl = uint32(c.MinTTLSeconds) //nolint:gosec // TTL bounds validated in config
	}

	t := max(c.MinTTLSeconds, min(int(ttl), c.MaxTTLSeconds))
	ttl = uint32(t) //nolint:gosec // TTL bounds validated in config

	// Calculate approximate size of the message
	itemSize := estimateMsgSize(msg)

	// Check if we need to evict entries due to size limit
	if c.MaxSizeBytes > 0 {
		c.sizeMu.Lock()
		// Remove old entry if exists (updating existing key)
		if oldItem, exists := c.lru.Peek(key); exists {
			c.currentSize -= oldItem.size
		}

		// Evict entries until we have enough space
		// Remove least recently used entries first
		evictedCount := 0

		for c.currentSize+itemSize > c.MaxSizeBytes && c.lru.Len() > 0 {
			// Get all keys and remove the first one (LRU order)
			keys := c.lru.Keys()
			if len(keys) == 0 {
				break
			}

			// Remove the least recently used entry
			oldKey := keys[0]
			if oldItem, exists := c.lru.Peek(oldKey); exists {
				c.lru.Remove(oldKey)
				c.currentSize -= oldItem.size
				evictedCount++
			} else {
				// Key was already removed, try next
				if len(keys) > 1 {
					oldKey = keys[1]
					if oldItem, exists := c.lru.Peek(oldKey); exists {
						c.lru.Remove(oldKey)
						c.currentSize -= oldItem.size
						evictedCount++
					}
				}

				break
			}
		}

		if evictedCount > 0 {
			// Log eviction at info level as it's important for debugging memory issues
			zerolog.Ctx(ctx).Info().
				Int("evicted_count", evictedCount).
				Int64("current_size_mb", c.currentSize/bytesPerMB).
				Int64("max_size_mb", c.MaxSizeBytes/bytesPerMB).
				Int("cache_entries", c.lru.Len()).
				Msg("cache entries evicted due to size limit")
		}

		c.currentSize += itemSize
		c.sizeMu.Unlock()
	}

	it := cacheItem{
		msg:    msg.Copy(),
		expire: time.Now().Add(time.Duration(ttl) * time.Second),
		size:   itemSize,
	}
	c.lru.Add(key, it)

	if cacheChangeNotify != nil {
		cacheChangeNotify()
	}
}

// estimateMsgSize estimates the size of a DNS message in bytes.
// Uses message serialization for accurate size calculation.
func estimateMsgSize(msg *dns.Msg) int64 {
	if msg == nil {
		return 0
	}

	// Serialize message to get accurate size
	packed, err := msg.Pack()
	if err != nil {
		// Fallback to approximation if packing fails
		return estimateMsgSizeApprox(msg)
	}

	return int64(len(packed))
}

// estimateMsgSizeApprox provides a fallback approximation if serialization fails.
func estimateMsgSizeApprox(msg *dns.Msg) int64 {
	if msg == nil {
		return 0
	}

	// Base message overhead (header ~12 bytes)
	size := int64(dnsHeaderSize)

	// Question section
	for _, q := range msg.Question {
		size += int64(len(q.Name)) + dnsQuestionOverhead // name + qtype + qclass
	}

	// Answer section - approximate 20-50 bytes per record
	size += int64(len(msg.Answer)) * avgDNSRecordSize

	// Authority section
	size += int64(len(msg.Ns)) * avgDNSRecordSize

	// Additional section
	size += int64(len(msg.Extra)) * avgDNSRecordSize

	return size
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

	if c.MaxSizeBytes > 0 {
		c.sizeMu.Lock()
		c.currentSize = 0
		c.sizeMu.Unlock()
	}

	if cacheChangeNotify != nil {
		cacheChangeNotify()
	}
}

// Delete removes cache entries for a specific name and qtype.
// If qtype is 0, deletes common types for the name.
//
//nolint:cyclop // complex branching for different qtypes
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
		if c.MaxSizeBytes > 0 {
			if item, exists := c.lru.Peek(key); exists {
				c.sizeMu.Lock()
				c.currentSize -= item.size
				c.sizeMu.Unlock()
			}
		}

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
		if c.MaxSizeBytes > 0 {
			if item, exists := c.lru.Peek(key); exists {
				c.sizeMu.Lock()
				c.currentSize -= item.size
				c.sizeMu.Unlock()
			}
		}

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

	end := min(offset+limit, len(tmp))

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

	if c.MaxSizeBytes > 0 {
		if item, exists := c.lru.Peek(key); exists {
			c.sizeMu.Lock()
			c.currentSize -= item.size
			c.sizeMu.Unlock()
		}
	}

	c.lru.Remove(key)

	if cacheChangeNotify != nil {
		cacheChangeNotify()
	}
}
