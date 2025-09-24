package dnsproxy

// UDPStrategy creates resolvers for UDP upstreams
type UDPStrategy struct{}

func (UDPStrategy) Supports(t string) bool { return t == "udp" }
func (UDPStrategy) NewResolver(t, address string, deps StrategyDeps) Resolver {
	return &UpstreamResolver{client: deps.UDP, network: t, address: address}
}
