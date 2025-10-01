// nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
package dnsproxy

import (
	"context"
	"math/rand" // nosemgrep: go.lang.security.audit.crypto.math_random.math-random-used
	"slices"

	"github.com/miekg/dns"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/lanresolver"
	"github.com/bavix/outway/internal/localzone"
)

//nolint:funlen
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

	// Get upstreams from manager
	rs := p.upstreams.RebuildResolvers(ctx, strategies, deps)

	// Ensure we have at least one resolver
	if len(rs) == 0 {
		// If no resolvers were created, create a default fallback resolver
		defaultUpstreams := []string{"udp:8.8.8.8:53", "udp:1.1.1.1:53"}
		for _, raw := range defaultUpstreams {
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
	}

	chain := NewChainResolver(rs...)

	// Create hosts resolver using manager
	cfg := p.config.GetConfig()
	hosts := p.hosts.CreateHostsResolver(chain, cfg)

	// Initialize zone detector and lease manager with auto-detection
	zoneDetector := localzone.NewZoneDetector()
	leaseManager := lanresolver.NewLeaseManager("/tmp/dhcp.leases")

	// Load initial leases
	if err := leaseManager.LoadLeases(); err != nil {
		// Log warning but continue
		// In a real implementation, you'd use proper logging
		_ = err // Suppress unused variable warning
	}

	// Create LAN resolver (always enabled)
	lanResolver := lanresolver.NewLANResolver(hosts, zoneDetector, leaseManager)
	next := Resolver(lanResolver)

	// Create mark resolver using managers
	mark := &MarkResolver{
		Next:    next,
		Backend: p.backend,
		Rules:   p.rules.GetRules(),
		Cfg:     cfg,
	}

	// Build core without metrics first so cache can wrap it
	var core Resolver = mark

	if cfg.Cache.Enabled && p.cache != nil {
		// Update the existing cache's Next resolver instead of creating a new one
		p.cache.UpdateCacheNext(core)

		if cfg.Cache.ServeStale {
			core = &ServeStaleResolver{Cache: p.cache.GetCache()}
		} else {
			core = p.cache.GetCache()
		}
	}

	// Place metrics outermost to include cache/hosts/upstreams in duration
	root := Resolver(&MetricsResolver{Next: core})
	p.active.Store(root)
}

func (p *Proxy) buildLegacyResolvers(strategies []UpstreamStrategy, deps StrategyDeps) []Resolver {
	var rs []Resolver

	upstreams := p.upstreams.GetUpstreamAddresses()
	for _, raw := range upstreams {
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
