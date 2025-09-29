// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
package dnsproxy

import (
	"context"
	"math/rand" // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"slices"

	"github.com/miekg/dns"

	"github.com/bavix/outway/internal/config"
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
		rs = p.buildWeightedResolvers(ups, strategies, deps)
	}

	chain := NewChainResolver(rs...)
	hosts := &HostsResolver{Next: chain, Hosts: p.cfg.Hosts, Cfg: p.cfg}
	mark := &MarkResolver{Next: hosts, Backend: p.backend, Rules: p.rules, Cfg: p.cfg}

	// Build core without metrics first so cache can wrap it
	var core Resolver = mark

	if p.cfg.Cache.Enabled {
		cache := NewCachedResolver(
			core,
			p.cfg.Cache.MaxEntries,
			p.cfg.Cache.MinTTLSeconds,
			p.cfg.Cache.MaxTTLSeconds,
		)
		if p.cfg.Cache.ServeStale {
			core = &ServeStaleResolver{Cache: cache}
		} else {
			core = cache
		}
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

// buildWeightedResolvers creates resolvers grouped by weight with random selection within each group.
func (p *Proxy) buildWeightedResolvers(ups []config.UpstreamConfig, strategies []UpstreamStrategy, deps StrategyDeps) []Resolver {
	// Copy and sort upstreams by weight desc using slices.SortFunc
	sorted := make([]config.UpstreamConfig, len(ups))
	copy(sorted, ups)
	slices.SortFunc(sorted, func(a, b config.UpstreamConfig) int {
		// desc
		if a.Weight == b.Weight {
			return 0
		}

		if a.Weight > b.Weight {
			return -1
		}

		return 1
	})

	// Build resolvers preserving shuffled order within equal weights
	var rs []Resolver

	i := 0
	for i < len(sorted) {
		j := i + 1
		for j < len(sorted) && sorted[j].Weight == sorted[i].Weight {
			j++
		}

		group := sorted[i:j]
		rs = append(rs, p.buildResolversFromGroup(group, strategies, deps)...)
		i = j
	}

	return rs
}

// sortWeightsDesc returns weights in descending order.
// sortWeightsDesc removed in favor of slices.SortFunc above

// buildResolversFromGroup creates resolvers from a weight group with random ordering.
func (p *Proxy) buildResolversFromGroup(group []config.UpstreamConfig, strategies []UpstreamStrategy, deps StrategyDeps) []Resolver {
	// Shuffle upstreams within the same weight group for random selection
	// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	rand.Shuffle(len(group), func(i, j int) {
		group[i], group[j] = group[j], group[i]
	})

	var rs []Resolver

	for _, u := range group {
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

	return rs
}
