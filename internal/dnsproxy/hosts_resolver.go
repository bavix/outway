package dnsproxy

import (
	"context"
	"net"
	"strings"

	"github.com/miekg/dns"

	"github.com/bavix/outway/internal/config"
)

// HostsResolver answers from static hosts list; falls through to Next if no match.
type HostsResolver struct {
	Next  Resolver
	Hosts []config.HostOverride
}

func (h *HostsResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if q == nil || len(q.Question) == 0 {
		return h.Next.Resolve(ctx, q)
	}
	name := strings.TrimSuffix(strings.ToLower(q.Question[0].Name), ".")
	qtype := q.Question[0].Qtype

	var aRecords []net.IP
	var aaaaRecords []net.IP
	ttl := uint32(60)
	for _, ho := range h.Hosts {
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

	// Build response
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
