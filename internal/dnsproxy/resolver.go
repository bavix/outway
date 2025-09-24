package dnsproxy

import (
	"context"
	"net/http"

	"github.com/miekg/dns"
)

// Resolver is a pluggable DNS resolution pipeline component.
type Resolver interface {
	Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error)
}

// UpstreamStrategy abstracts building resolvers for specific upstream types
type UpstreamStrategy interface {
	Supports(t string) bool
	NewResolver(t string, address string, deps StrategyDeps) Resolver
}

// StrategyDeps provides dependencies to build resolvers
type StrategyDeps struct {
	UDP         *dns.Client
	TCP         *dns.Client
	DoH         *http.Client
	ExchangeDoH func(msg *dns.Msg, url string) (*dns.Msg, error)
}
