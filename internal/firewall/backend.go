//nolint:ireturn
package firewall

import (
	"context"
	"errors"
	"runtime"

	"github.com/rs/zerolog"
)

var errNoSupportedFirewallBackendDetected = errors.New("no supported firewall backend detected")

// TunnelInfo represents information about a tunnel interface.
type TunnelInfo struct {
	Name     string
	TableID  int
	FwMark   int
	Priority int
}

// Backend interface for firewall operations.
type Backend interface {
	Name() string
	EnsurePolicy(ctx context.Context, iface string) error
	MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error
	CleanupAll(ctx context.Context) error
	// New methods for dynamic table management
	InitializeTunnels(ctx context.Context, tunnels []string) ([]TunnelInfo, error)
	FlushRuntime(ctx context.Context) error
	GetTunnelInfo(ctx context.Context, iface string) (*TunnelInfo, error)
}

// DetectBackend detects the appropriate firewall backend for the current system.
func DetectBackend(ctx context.Context) (Backend, error) { //nolint:ireturn
	log := zerolog.Ctx(ctx)

	switch runtime.GOOS {
	case "linux":
		if b, err := NewRouteBackend(); err == nil && b != nil {
			log.Info().Str("backend", b.Name()).Msg("firewall backend selected")

			return b, nil
		}

		if b := NewIPTablesBackend(); b != nil {
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
