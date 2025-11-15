package dnsproxy

import (
	"context"
	"errors"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
)

var errInvalidUpstreamClientOrQuery = errors.New("invalid upstream client or query")

// UpstreamResolver talks to a single upstream (udp/tcp or via exchange func).
type UpstreamResolver struct {
	client   *dns.Client
	network  string
	address  string
	exchange func(*dns.Msg, string) (*dns.Msg, error)
}

//nolint:cyclop
func (u *UpstreamResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if u.exchange != nil {
		if out, err := u.exchange(q, u.address); err == nil && out != nil {
			return out, u.network + ":" + u.address, nil
		} else {
			zerolog.Ctx(ctx).Err(err).Str("net", u.network).Str("upstream", u.address).Msg("dns upstream doh error")

			return nil, u.network + ":" + u.address, err
		}
	}

	if u.client == nil || q == nil {
		return nil, u.network + ":" + u.address, errInvalidUpstreamClientOrQuery
	}

	out, _, err := u.client.Exchange(q, u.address)
	// If UDP response is truncated, treat as error to allow next strategy (e.g., TCP) to retry
	if err == nil && out != nil && out.Truncated {
		err = errors.New("truncated") //nolint:err113

		zerolog.Ctx(ctx).Debug().
			Str("net", u.network).
			Str("upstream", u.address).
			Msg("DNS response truncated, will retry with TCP")
	}

	if err != nil || out == nil {
		queryName := ""
		if len(q.Question) > 0 {
			queryName = q.Question[0].Name
		}

		zerolog.Ctx(ctx).Err(err).
			Str("net", u.network).
			Str("upstream", u.address).
			Str("query", queryName).
			Msg("dns upstream error")
	} else if len(q.Question) > 0 {
		// Log successful resolution at debug level
		zerolog.Ctx(ctx).Debug().
			Str("net", u.network).
			Str("upstream", u.address).
			Str("query", q.Question[0].Name).
			Int("answers", len(out.Answer)).
			Msg("dns upstream resolved successfully")
	}

	return out, u.network + ":" + u.address, err
}
