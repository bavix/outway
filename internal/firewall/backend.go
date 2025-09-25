package firewall

import (
	"context"
	"errors"
	"runtime"

	"github.com/rs/zerolog"
)

var errNoSupportedFirewallBackendDetected = errors.New("no supported firewall backend detected")

type Backend interface {
	Name() string
	EnsurePolicy(ctx context.Context, iface string) error
	MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error
	CleanupAll(ctx context.Context) error
}

//nolint:ireturn
func DetectBackend(ctx context.Context) (Backend, error) {
	log := zerolog.Ctx(ctx)

	switch runtime.GOOS {
	case "linux":
		if b := newNFTBackend(ctx); b != nil {
			log.Info().Str("backend", b.Name()).Msg("firewall backend selected")

			return b, nil
		}

		if b := newIPTablesBackend(); b != nil {
			log.Info().Str("backend", b.Name()).Msg("firewall backend selected")

			return b, nil
		}
	case "darwin":
		if b := newPFBackend(); b != nil {
			log.Info().Str("backend", b.Name()).Msg("firewall backend selected")

			return b, nil
		}
	}

	return nil, errNoSupportedFirewallBackendDetected
}
