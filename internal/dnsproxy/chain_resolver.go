package dnsproxy

import (
	"context"
	"errors"

	"github.com/miekg/dns"
)

type ChainResolver struct{ resolvers []Resolver }

func NewChainResolver(rs ...Resolver) *ChainResolver { return &ChainResolver{resolvers: rs} }

func (c *ChainResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if len(c.resolvers) == 0 {
		return nil, "", errors.New("no upstreams configured")
	}
	var firstErr error
	for _, r := range c.resolvers {
		if r == nil {
			continue
		}
		out, src, err := r.Resolve(ctx, q)
		if err == nil && out != nil {
			return out, src, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr == nil {
		firstErr = errors.New("all upstreams failed")
	}
	return nil, "", firstErr
}

// strategies moved to separate files
