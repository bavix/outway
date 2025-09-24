package dnsproxy

import (
	"fmt"
	"strings"

	"github.com/bavix/outway/internal/config"
	"github.com/miekg/dns"
)

func (p *Proxy) rebuildResolver() {
	// Build upstream resolvers using weighted order from config
	strategies := []UpstreamStrategy{UDPStrategy{}, TCPStrategy{}, DoHStrategy{}, DotStrategy{}}
	deps := StrategyDeps{
		UDP: p.dnsUDP,
		TCP: p.dnsTCP,
		DoH: p.dohClient,
		ExchangeDoH: func(m *dns.Msg, url string) (*dns.Msg, error) {
			out, _, err := p.exchangeDoH(m, url)
			return out, err
		},
	}
	var rs []Resolver
	// Prefer explicit config objects (with weight)
	ups := p.cfg.GetUpstreamsByWeight()
	if len(ups) == 0 {
		// Fallback to legacy string list
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
		max := p.cfg.Cache.MaxEntries
		if max <= 0 {
			max = 10000
		}
		core = NewCachedResolver(core, max)
	}

	// Place metrics outermost to include cache/hosts/upstreams in duration
	root := Resolver(&MetricsResolver{Next: core})
	p.active.Store(root)
}

// Utility to parse single upstream string from config.Upstreams when live update called via admin API
func toUpstreamConfig(s string, index int) (config.UpstreamConfig, error) {
	if strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "http://") {
		return config.UpstreamConfig{Name: fmt.Sprintf("DoH-%d", index+1), Address: s, Type: "doh"}, nil
	}
	parts := strings.SplitN(s, ":", 3)
	if len(parts) != 3 || (parts[0] != "udp" && parts[0] != "tcp") || parts[1] == "" || parts[2] == "" {
		return config.UpstreamConfig{}, fmt.Errorf("invalid upstream: %s", s)
	}
	return config.UpstreamConfig{Name: fmt.Sprintf("%s-%s", parts[0], parts[1]), Address: parts[1] + ":" + parts[2], Type: parts[0]}, nil
}
