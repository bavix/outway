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

func (u *UpstreamResolver) Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	if u.exchange != nil {
		if out, err := u.exchange(q, u.address); err == nil && out != nil {
			return out, u.network + ":" + u.address, nil
		} else {
			zerolog.Ctx(ctx).Error().Err(err).Str("net", u.network).Str("upstream", u.address).Msg("dns upstream doh error")

			return nil, u.network + ":" + u.address, err
		}
	}

	if u.client == nil || q == nil {
		return nil, u.network + ":" + u.address, errInvalidUpstreamClientOrQuery
	}

	out, _, err := u.client.Exchange(q, u.address)
	if err != nil || out == nil {
		zerolog.Ctx(ctx).Error().Err(err).Str("net", u.network).Str("upstream", u.address).Msg("dns upstream error")
	}

	return out, u.network + ":" + u.address, err
}
