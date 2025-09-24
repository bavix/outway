package firewall

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type iptablesBackend struct {
	mu     sync.Mutex
	ifaces map[string]struct{}
}

func newIPTablesBackend() Backend {
	if _, err := exec.LookPath("iptables"); err != nil {
		return nil
	}

	return &iptablesBackend{ifaces: make(map[string]struct{})}
}

func (b *iptablesBackend) Name() string { return "iptables" }

func (b *iptablesBackend) EnsurePolicy(ctx context.Context, iface string) error {
	zerolog.Ctx(ctx).Info().Str("iface", iface).Msg("ensure iptables policy")
	_ = exec.Command("ipset", "create", setName4(iface), "hash:ip", "timeout", "0").Run()
	_ = exec.Command("ipset", "create", setName6(iface), "hash:ip", "family", "inet6", "timeout", "0").Run()

	b.mu.Lock()
	b.ifaces[iface] = struct{}{}
	b.mu.Unlock()

	return nil
}

func (b *iptablesBackend) MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error {
	if ttlSeconds < 30 {
		ttlSeconds = 30
	}

	zerolog.Ctx(ctx).Debug().Str("iface", iface).Str("ip", ip).Int("ttl", ttlSeconds).Msg("mark ip")
	set := setName4(iface)

	cmd := exec.Command("ipset", "add", set, ip, "timeout", strconv.Itoa(ttlSeconds), "-exist")
	if isIPv6(ip) {
		set = setName6(iface)
		cmd = exec.Command("ipset", "add", set, ip, "timeout", strconv.Itoa(ttlSeconds), "-exist")
	}

	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Str("out", string(out)).Msg("ipset add failed")
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
		_ = exec.Command("ipset", "destroy", setName4(iface)).Run()
		_ = exec.Command("ipset", "destroy", setName6(iface)).Run()
	}

	return nil
}

func setName4(iface string) string { return fmt.Sprintf("outway_%s_4", iface) }
func setName6(iface string) string { return fmt.Sprintf("outway_%s_6", iface) }

var _ = time.Second
