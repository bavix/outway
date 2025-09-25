package dnsproxy

import (
	"context"
	"math/rand"

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

// buildWeightedResolvers creates resolvers grouped by weight with random selection within each group.
func (p *Proxy) buildWeightedResolvers(ups []config.UpstreamConfig, strategies []UpstreamStrategy, deps StrategyDeps) []Resolver {
	// Group upstreams by weight
	weightGroups := make(map[int][]config.UpstreamConfig)
	for _, u := range ups {
		weightGroups[u.Weight] = append(weightGroups[u.Weight], u)
	}

	// Process weight groups in descending order
	weights := p.sortWeightsDesc(weightGroups)

	var rs []Resolver

	for _, weight := range weights {
		group := weightGroups[weight]
		rs = append(rs, p.buildResolversFromGroup(group, strategies, deps)...)
	}

	return rs
}

// sortWeightsDesc returns weights in descending order.
func (p *Proxy) sortWeightsDesc(weightGroups map[int][]config.UpstreamConfig) []int {
	weights := make([]int, 0, len(weightGroups))
	for weight := range weightGroups {
		weights = append(weights, weight)
	}

	// Sort weights in descending order
	for i := 0; i < len(weights); i++ {
		for j := i + 1; j < len(weights); j++ {
			if weights[j] > weights[i] {
				weights[i], weights[j] = weights[j], weights[i]
			}
		}
	}

	return weights
}

// buildResolversFromGroup creates resolvers from a weight group with random ordering.
func (p *Proxy) buildResolversFromGroup(group []config.UpstreamConfig, strategies []UpstreamStrategy, deps StrategyDeps) []Resolver {
	// Shuffle upstreams within the same weight group for random selection
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
