package dnsproxy

import (
	"context"
	"errors"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
)

var (
	errNoUpstreamsConfigured = errors.New("no upstreams configured")
	errAllUpstreamsFailed    = errors.New("all upstreams failed")
)

type ChainResolver struct{ resolvers []Resolver }

func NewChainResolver(rs ...Resolver) *ChainResolver { return &ChainResolver{resolvers: rs} }

//nolint:cyclop,funlen // complex fallback logic through multiple resolvers
func (c *ChainResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if len(c.resolvers) == 0 {
		return nil, "", errNoUpstreamsConfigured
	}

	var (
		firstErr     error
		lastEmptyOut *dns.Msg
		lastEmptySrc string
	)

	for i, r := range c.resolvers {
		if r == nil {
			continue
		}

		out, src, err := r.Resolve(ctx, q)
		if err == nil && out != nil {
			if len(out.Answer) > 0 {
				// Log successful resolution at debug level
				if len(q.Question) > 0 {
					zerolog.Ctx(ctx).Debug().
						Str("query", q.Question[0].Name).
						Int("resolver_index", i).
						Str("upstream", src).
						Int("answers", len(out.Answer)).
						Msg("chain resolver: upstream succeeded")
				}

				return out, src, nil
			}
			// keep last empty response (NOERROR/NODATA)
			lastEmptyOut = out
			lastEmptySrc = src

			zerolog.Ctx(ctx).Debug().
				Int("resolver_index", i).
				Str("upstream", src).
				Msg("chain resolver: upstream returned empty answer, trying next")

			continue
		}

		// Log upstream failure at debug level
		if len(q.Question) > 0 {
			zerolog.Ctx(ctx).Debug().
				Err(err).
				Int("resolver_index", i).
				Str("upstream", src).
				Str("query", q.Question[0].Name).
				Msg("chain resolver: upstream failed, trying next")
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
