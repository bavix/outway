package dnsproxy

// TCPStrategy creates resolvers for TCP upstreams
type TCPStrategy struct{}

func (TCPStrategy) Supports(t string) bool { return t == "tcp" }
func (TCPStrategy) NewResolver(t, address string, deps StrategyDeps) Resolver {
	return &UpstreamResolver{client: deps.TCP, network: t, address: address}
}
