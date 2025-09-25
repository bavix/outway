package dnsproxy

import (
	"context"

	"github.com/miekg/dns"
)

func (p *Proxy) rebuildResolver(ctx context.Context) {
	// Build upstream resolvers using weighted order from config
	strategies := []UpstreamStrategy{UDPStrategy{}, TCPStrategy{}, DoHStrategy{}, DotStrategy{}}
	deps := StrategyDeps{
		UDP: p.dnsUDP,
		TCP: p.dnsTCP,
		DoH: p.dohClient,
		ExchangeDoH: func(m *dns.Msg, url string) (*dns.Msg, error) {
			out, _, err := p.exchangeDoH(ctx, m, url)

			return out, err
		},
	}

	var rs []Resolver
	// Prefer explicit config objects (with weight)
	ups := p.cfg.GetUpstreamsByWeight()
	if len(ups) == 0 {
		// Fallback to legacy string list
		rs = p.buildLegacyResolvers(strategies, deps)
	} else {
		for _, u := range ups { // already sorted desc by weight
			netw, addr := u.Type, u.Address
			for _, s := range strategies {
				if s.Supports(netw) {
					if r := s.NewResolver(netw, addr, deps); r != nil {
						rs = append(rs, r)
					}

					break
				}
			}
		}
	}

	chain := NewChainResolver(rs...)
	hosts := &HostsResolver{Next: chain, Hosts: p.cfg.Hosts}
	mark := &MarkResolver{Next: hosts, Backend: p.backend, Rules: p.rules, Cfg: p.cfg}

	// Build core without metrics first so cache can wrap it
	var core Resolver = mark

	if p.cfg.Cache.Enabled {
		maxEntries := p.cfg.Cache.MaxEntries
		if maxEntries <= 0 {
			maxEntries = 10000
		}

		core = NewCachedResolver(core, maxEntries)
	}

	// Place metrics outermost to include cache/hosts/upstreams in duration
	root := Resolver(&MetricsResolver{Next: core})
	p.active.Store(root)
}

func (p *Proxy) buildLegacyResolvers(strategies []UpstreamStrategy, deps StrategyDeps) []Resolver {
	var rs []Resolver

	for _, raw := range p.upstreams {
		netw, addr := parseUpstream(raw)
		for _, s := range strategies {
			if s.Supports(netw) {
				if r := s.NewResolver(netw, addr, deps); r != nil {
					rs = append(rs, r)
				}

				break
			}
		}
	}

	return rs
}
