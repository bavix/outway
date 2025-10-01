package lanresolver

import (
	"context"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/dnsproxy"
)

const (
	defaultTTL = 60
)

// Resolver resolves DNS queries for local zones from DHCP leases.
type Resolver struct {
	Next     dnsproxy.Resolver
	zones    []string
	hostMap  map[string][]string
	leases   []Lease
	mu       sync.RWMutex
	cfg      *config.Config
	onChange func() // Callback for UI updates
}

// NewResolver creates a new LAN resolver.
func NewResolver(next dnsproxy.Resolver, zones []string, cfg *config.Config) *Resolver {
	return &Resolver{
		Next:    next,
		zones:   zones,
		hostMap: make(map[string][]string),
		cfg:     cfg,
	}
}

// UpdateLeases updates the lease data and rebuilds the host map.
func (r *Resolver) UpdateLeases(leases []Lease, zones []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.leases = leases
	r.zones = zones
	r.hostMap = BuildHostMap(leases, zones)

	// Trigger onChange callback if set
	if r.onChange != nil {
		r.onChange()
	}
}

// SetOnChange registers a callback for lease updates.
func (r *Resolver) SetOnChange(callback func()) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onChange = callback
}

// GetZones returns the current local zones.
func (r *Resolver) GetZones() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.zones
}

// GetLeases returns the current leases.
func (r *Resolver) GetLeases() []Lease {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.leases
}

// Resolve implements the dnsproxy.Resolver interface.
func (r *Resolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if q == nil || len(q.Question) == 0 {
		return r.Next.Resolve(ctx, q)
	}

	name := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(q.Question[0].Name, ".")))
	qtype := q.Question[0].Qtype

	// Check if this query is for a local zone
	if !r.isLocalZone(name) {
		return r.Next.Resolve(ctx, q)
	}

	// Look up hostname in host map
	r.mu.RLock()
	ips, found := r.hostMap[name]
	r.mu.RUnlock()

	if !found {
		// Return NXDOMAIN for local zones not found in leases
		m := new(dns.Msg)
		m.SetRcode(q, dns.RcodeNameError)
		m.Authoritative = true

		zerolog.Ctx(ctx).Debug().
			Str("name", name).
			Msg("local zone query: NXDOMAIN")

		return m, "lan-resolver", nil
	}

	// Build response with matching records
	m := new(dns.Msg)
	m.SetReply(q)
	m.Authoritative = true

	ttl := uint32(defaultTTL)
	if r.cfg != nil && r.cfg.Cache.Enabled {
		ttl = uint32(max(r.cfg.Cache.MinTTLSeconds, min(int(ttl), r.cfg.Cache.MaxTTLSeconds)))
	}

	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}

		// Add A record for IPv4
		if qtype == dns.TypeA || qtype == dns.TypeANY {
			if v4 := ip.To4(); v4 != nil {
				rr := &dns.A{
					Hdr: dns.RR_Header{
						Name:   q.Question[0].Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    ttl,
					},
					A: v4,
				}
				m.Answer = append(m.Answer, rr)
			}
		}

		// Add AAAA record for IPv6
		if qtype == dns.TypeAAAA || qtype == dns.TypeANY {
			if v6 := ip.To16(); v6 != nil && ip.To4() == nil {
				rr := &dns.AAAA{
					Hdr: dns.RR_Header{
						Name:   q.Question[0].Name,
						Rrtype: dns.TypeAAAA,
						Class:  dns.ClassINET,
						Ttl:    ttl,
					},
					AAAA: v6,
				}
				m.Answer = append(m.Answer, rr)
			}
		}
	}

	// If no matching record type found, return empty response
	if len(m.Answer) == 0 {
		m.Rcode = dns.RcodeSuccess
	}

	zerolog.Ctx(ctx).Debug().
		Str("name", name).
		Int("answers", len(m.Answer)).
		Msg("local zone query resolved")

	return m, "lan-resolver", nil
}

// isLocalZone checks if the query name ends with a local zone.
func (r *Resolver) isLocalZone(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, zone := range r.zones {
		if name == zone || strings.HasSuffix(name, "."+zone) {
			return true
		}
	}

	// Also check bare hostname (without zone)
	_, found := r.hostMap[name]
	return found
}
