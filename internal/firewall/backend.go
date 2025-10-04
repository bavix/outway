package firewall

import (
	"context"
	"errors"
	"runtime"

	"github.com/rs/zerolog"
)

var errNoSupportedFirewallBackendDetected = errors.New("no supported firewall backend detected")

// Backend interface for firewall operations.
type Backend interface {
	Name() string
	MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error
	CleanupAll(ctx context.Context) error
}

// DetectBackend detects the appropriate firewall backend for the current system.
//
//nolint:ireturn
func DetectBackend(ctx context.Context) (Backend, error) {
	log := zerolog.Ctx(ctx)

	switch runtime.GOOS {
	case "linux":
		// Use simple route backend (uses ip route expires for automatic cleanup)
		if b := NewSimpleRouteBackend(); b != nil {
			log.Info().Str("backend", b.Name()).Msg("firewall backend selected")

			return b, nil
		}
	case "darwin":
		if b := NewPFBackend(); b != nil {
			log.Info().Str("backend", b.Name()).Msg("firewall backend selected")

			return b, nil
		}
	}

	return nil, errNoSupportedFirewallBackendDetected
}
