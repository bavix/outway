package dnsproxy

import (
	"context"
	"errors"

	"github.com/miekg/dns"
)

var (
	errNoUpstreamsConfigured = errors.New("no upstreams configured")
	errAllUpstreamsFailed    = errors.New("all upstreams failed")
)

type ChainResolver struct{ resolvers []Resolver }

func NewChainResolver(rs ...Resolver) *ChainResolver { return &ChainResolver{resolvers: rs} }

func (c *ChainResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if len(c.resolvers) == 0 {
		return nil, "", errNoUpstreamsConfigured
	}

    var firstErr error
    var lastEmptyOut *dns.Msg
    var lastEmptySrc string

	for _, r := range c.resolvers {
		if r == nil {
			continue
		}

        out, src, err := r.Resolve(ctx, q)
        if err == nil && out != nil {
            if len(out.Answer) > 0 {
                return out, src, nil
            }
            // keep last empty response (NOERROR/NODATA)
            lastEmptyOut = out
            lastEmptySrc = src
            continue
        }

		if firstErr == nil {
			firstErr = err
		}
	}

    if lastEmptyOut != nil {
        return lastEmptyOut, lastEmptySrc, nil
    }

    if firstErr == nil {
        firstErr = errAllUpstreamsFailed
    }

    return nil, "", firstErr
}

// strategies moved to separate files
