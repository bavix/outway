package dnsproxy

// TCPStrategy creates resolvers for TCP upstreams.
type TCPStrategy struct{}

func (TCPStrategy) Supports(t string) bool { return t == protocolTCP }
func (TCPStrategy) NewResolver(t, address string, deps StrategyDeps) Resolver { //nolint:ireturn
	return &UpstreamResolver{client: deps.TCP, network: t, address: address}
}
