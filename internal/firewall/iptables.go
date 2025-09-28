package firewall

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const (
	minTTLSeconds = 30
)

type iptablesBackend struct {
	mu     sync.Mutex
	ifaces map[string]struct{}
}

func NewIPTablesBackend() *iptablesBackend {
	if _, err := exec.LookPath("iptables"); err != nil {
		return nil
	}

	return &iptablesBackend{ifaces: make(map[string]struct{})}
}

func (b *iptablesBackend) Name() string { return "iptables" }

func (b *iptablesBackend) EnsurePolicy(ctx context.Context, iface string) error {
	zerolog.Ctx(ctx).Info().Str("iface", iface).Msg("ensure iptables policy")
	_ = exec.CommandContext(ctx, "ipset", "create", SetName4(iface), "hash:ip", "timeout", "0").Run()                    //nolint:gosec
	_ = exec.CommandContext(ctx, "ipset", "create", SetName6(iface), "hash:ip", "family", "inet6", "timeout", "0").Run() //nolint:gosec

	b.mu.Lock()
	b.ifaces[iface] = struct{}{}
	b.mu.Unlock()

	return nil
}

func (b *iptablesBackend) MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error {
	if ttlSeconds < minTTLSeconds {
		ttlSeconds = minTTLSeconds
	}

	zerolog.Ctx(ctx).Debug().Str("iface", iface).IPAddr("ip", net.ParseIP(ip)).Int("ttl", ttlSeconds).Msg("mark ip")
	set := SetName4(iface)

	cmd := exec.CommandContext(ctx, "ipset", "add", set, ip, "timeout", strconv.Itoa(ttlSeconds), "-exist") //nolint:gosec
	if IsIPv6(ip) {
		set = SetName6(iface)
		cmd = exec.CommandContext(ctx, "ipset", "add", set, ip, "timeout", strconv.Itoa(ttlSeconds), "-exist") //nolint:gosec
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("out", out).Msg("ipset add failed")

		return fmt.Errorf("failed to add IP %s to ipset %s: %w", ip, set, err)
	}

	return nil
}

func (b *iptablesBackend) CleanupAll(ctx context.Context) error {
	zerolog.Ctx(ctx).Info().Msg("cleanup iptables/ipset sets")
	b.mu.Lock()

	ifaces := make([]string, 0, len(b.ifaces))
	for k := range b.ifaces {
		ifaces = append(ifaces, k)
	}

	b.mu.Unlock()

	for _, iface := range ifaces {
		_ = exec.CommandContext(ctx, "ipset", "destroy", SetName4(iface)).Run() //nolint:gosec // ipset is a system utility
		_ = exec.CommandContext(ctx, "ipset", "destroy", SetName6(iface)).Run() //nolint:gosec // ipset is a system utility
	}

	return nil
}

// InitializeTunnels is a no-op for iptables backend.
func (b *iptablesBackend) InitializeTunnels(ctx context.Context, tunnels []string) ([]TunnelInfo, error) {
	// iptables backend doesn't use dynamic tables
	return []TunnelInfo{}, nil
}

// FlushRuntime flushes runtime data for iptables backend.
func (b *iptablesBackend) FlushRuntime(ctx context.Context) error {
	zerolog.Ctx(ctx).Info().Msg("flushing iptables runtime data")
	// For iptables, we just flush the sets
	return b.CleanupAll(ctx)
}

// GetTunnelInfo returns nil for iptables backend.
func (b *iptablesBackend) GetTunnelInfo(ctx context.Context, iface string) (*TunnelInfo, error) {
	// iptables backend doesn't use dynamic tables
	return nil, fmt.Errorf("%w: iptables backend", ErrTunnelNotFound)
}

func SetName4(iface string) string { return fmt.Sprintf("outway_%s_4", iface) }
func SetName6(iface string) string { return fmt.Sprintf("outway_%s_6", iface) }

var _ = time.Second
