package dnsproxy

// TCPStrategy creates resolvers for TCP upstreams.
type TCPStrategy struct{}

func (TCPStrategy) Supports(t string) bool { return t == protocolTCP }
func (TCPStrategy) NewResolver(t, address string, deps StrategyDeps) *UpstreamResolver {
	return &UpstreamResolver{client: deps.TCP, network: t, address: address}
}
