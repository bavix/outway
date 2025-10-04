package dnsproxy

import "github.com/miekg/dns"

// DoHStrategy creates resolvers for DNS-over-HTTPS upstreams.
type DoHStrategy struct{}

func (DoHStrategy) Supports(t string) bool { return t == "doh" }
func (DoHStrategy) NewResolver(t, address string, deps StrategyDeps) *UpstreamResolver {
	exch := func(m *dns.Msg, url string) (*dns.Msg, error) { return deps.ExchangeDoH(m, url) }

	return &UpstreamResolver{network: t, address: address, exchange: exch}
}
