package dnsproxy

import (
	"context"
	"net"
	"strings"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"

	"github.com/bavix/outway/internal/config"
)

const (
	defaultHostsTTL = 60
)

// HostsResolver answers from static hosts list; falls through to Next if no match.
// If HostsManager is provided, it reads hosts dynamically from manager (thread-safe).
// Otherwise, uses static Hosts slice.
type HostsResolver struct {
	Next         Resolver
	Hosts        []config.HostOverride // Static hosts (used if HostsManager is nil)
	HostsManager HostsManager          // Dynamic hosts manager (optional)
	Cfg          *config.Config
}

func (h *HostsResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) { //nolint:gocognit,cyclop,funlen
	if q == nil || len(q.Question) == 0 {
		return h.Next.Resolve(ctx, q)
	}

	name := strings.ToLower(strings.TrimSpace(strings.TrimSuffix(q.Question[0].Name, ".")))
	qtype := q.Question[0].Qtype

	// Get hosts dynamically from manager if available, otherwise use static hosts
	var hosts []config.HostOverride
	if h.HostsManager != nil {
		hosts = h.HostsManager.GetHosts()
	} else {
		hosts = h.Hosts
	}

	var (
		aRecords    []net.IP
		aaaaRecords []net.IP
	)

	ttl := uint32(defaultHostsTTL)

	for _, ho := range hosts {
		if matchDomainPattern(ho.Pattern, name) {
			for _, s := range ho.A {
				if ip := net.ParseIP(s); ip != nil {
					aRecords = append(aRecords, ip)
				}
			}

			for _, s := range ho.AAAA {
				if ip := net.ParseIP(s); ip != nil {
					aaaaRecords = append(aaaaRecords, ip)
				}
			}

			if ho.TTL > 0 {
				ttl = ho.TTL
			}

			break
		}
	}

	if len(aRecords) == 0 && len(aaaaRecords) == 0 {
		return h.Next.Resolve(ctx, q)
	}

	// Log hosts match at debug level
	zerolog.Ctx(ctx).Debug().
		Str("query", name).
		Uint16("qtype", qtype).
		Int("a_records", len(aRecords)).
		Int("aaaa_records", len(aaaaRecords)).
		Uint32("ttl", ttl).
		Msg("hosts override matched")

	// Clamp TTL to configured cache bounds if available
	if h.Cfg != nil && h.Cfg.Cache.Enabled {
		ttl = uint32(max(h.Cfg.Cache.MinTTLSeconds, min(int(ttl), h.Cfg.Cache.MaxTTLSeconds))) //nolint:gosec // TTL bounds validated in config
	}

	m := new(dns.Msg)
	m.SetReply(q)
	m.Authoritative = true

	if qtype == dns.TypeA || qtype == dns.TypeANY {
		for _, ip := range aRecords {
			if v4 := ip.To4(); v4 != nil {
				rr := &dns.A{Hdr: dns.RR_Header{Name: q.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl}, A: v4}
				m.Answer = append(m.Answer, rr)
			}
		}
	}

	if qtype == dns.TypeAAAA || qtype == dns.TypeANY {
		for _, ip := range aaaaRecords {
			if v6 := ip.To16(); v6 != nil && ip.To4() == nil {
				rr := &dns.AAAA{Hdr: dns.RR_Header{Name: q.Question[0].Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl}, AAAA: v6}
				m.Answer = append(m.Answer, rr)
			}
		}
	}

	return m, "hosts", nil
}
