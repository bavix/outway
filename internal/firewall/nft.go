package firewall

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type nftBackend struct {
	mutex    sync.Mutex
	appTable string
	entries  map[string]time.Time // ip -> expireAt
}

func newNFTBackend() Backend {
	if _, err := exec.LookPath("nft"); err != nil {
		return nil
	}

	return &nftBackend{
		appTable: "outway",
		entries:  make(map[string]time.Time),
	}
}

func (n *nftBackend) Name() string { return "nftables" }

func (n *nftBackend) EnsurePolicy(ctx context.Context, iface string) error {
	zerolog.Ctx(ctx).Info().Str("iface", iface).Msg("ensure nft policy")
	// Create table and set if not exist
	_ = exec.Command("nft", "add", "table", "inet", n.appTable).Run()
	_ = exec.Command("nft", "add", "set", "inet", n.appTable, ifaceSet(iface), "{ type ipv4_addr; flags timeout; } ").Run()
	_ = exec.Command("nft", "add", "set", "inet", n.appTable, ifaceSet6(iface), "{ type ipv6_addr; flags timeout; } ").Run()

	return nil
}

func (n *nftBackend) MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	if ttlSeconds < 30 {
		ttlSeconds = 30
	}

	zerolog.Ctx(ctx).Debug().Str("iface", iface).Str("ip", ip).Int("ttl", ttlSeconds).Msg("mark ip")

	set := ifaceSet(ipvFamilyFromIP(ip, iface))
	if isIPv6(ip) {
		set = ifaceSet6(ipvFamilyFromIP(ip, iface))
	}

	cmd := exec.Command("nft", "add", "element", "inet", n.appTable, set, fmt.Sprintf("{ %s timeout %ds }", ip, ttlSeconds))
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Str("out", string(out)).Msg("nft add element failed")
	}
	// Track expiry; best-effort cleanup timer
	exp := time.Now().Add(time.Duration(ttlSeconds) * time.Second)

	n.entries[ip] = exp
	go n.expireAfter(ip, exp)

	return nil
}

func (n *nftBackend) CleanupAll(ctx context.Context) error {
	zerolog.Ctx(ctx).Info().Str("table", n.appTable).Msg("cleanup nft table")
	cmd := exec.Command("nft", "delete", "table", "inet", n.appTable)
	_ = cmd.Run()

	return nil
}

func (n *nftBackend) expireAfter(ip string, expireAt time.Time) {
	t := time.NewTimer(time.Until(expireAt))
	<-t.C

	n.mutex.Lock()
	defer n.mutex.Unlock()
	// best effort remove (safe even if already expired)
	set := ifaceSet("")
	if isIPv6(ip) {
		set = ifaceSet6("")
	}

	_ = exec.Command("nft", "delete", "element", "inet", n.appTable, set, fmt.Sprintf("{ %s }", ip)).Run()
	delete(n.entries, ip)
}

func ifaceSet(iface string) string  { return iface + "_v4" }
func ifaceSet6(iface string) string { return iface + "_v6" }

func isIPv6(ip string) bool {
	for i := range len(ip) {
		if ip[i] == ':' {
			return true
		}
	}

	return false
}

func ipvFamilyFromIP(ip, iface string) string { return iface }
