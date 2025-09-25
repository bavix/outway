package dnsproxy

import (
	"context"
	"strings"

	"github.com/miekg/dns"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/firewall"
	"github.com/bavix/outway/internal/metrics"
)

type MarkResolver struct {
	Next    Resolver
	Backend firewall.Backend
	Rules   *RuleStore
	Cfg     *config.Config
}

func (m *MarkResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) { //nolint:cyclop
	out, src, err := m.Next.Resolve(ctx, q)
	if err != nil || out == nil || len(out.Answer) == 0 || m.Backend == nil || m.Rules == nil || q == nil || len(q.Question) == 0 {
		return out, src, err
	}

	name := strings.ToLower(strings.TrimSuffix(q.Question[0].Name, "."))

	rule, ok := m.Rules.Find(name)
	if !ok {
		return out, src, err
	}

	for _, rr := range out.Answer {
		switch a := rr.(type) {
		case *dns.A:
			ttl := a.Hdr.Ttl
			if rule.PinTTL {
				ttl = uint32(m.Cfg.GetMinMarkTTL(ttl).Seconds())
			} else {
				ttl = minTTL(ttl)
			}

			if err2 := m.Backend.MarkIP(ctx, rule.Via, a.A.String(), int(ttl)); err2 != nil {
				metrics.M.DNSMarksError.Inc()
			} else {
				metrics.M.DNSMarksSuccess.Inc()
			}
		case *dns.AAAA:
			ttl := a.Hdr.Ttl
			if rule.PinTTL {
				ttl = uint32(m.Cfg.GetMinMarkTTL(ttl).Seconds())
			} else {
				ttl = minTTL(ttl)
			}

			if err2 := m.Backend.MarkIP(ctx, rule.Via, a.AAAA.String(), int(ttl)); err2 != nil {
				metrics.M.DNSMarksError.Inc()
			} else {
				metrics.M.DNSMarksSuccess.Inc()
			}
		}
	}

	return out, src, err
}
