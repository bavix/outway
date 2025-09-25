package firewall

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type nftBackend struct {
	mutex    sync.Mutex
	appTable string
	entries  map[string]nftEntry // ip -> entry info
}

type nftEntry struct {
	expireAt time.Time
	iface    string
}

var (
	ErrInvalidIface = errors.New("invalid interface name")
	ErrInvalidIP    = errors.New("invalid IP address")
)

func newNFTBackend() *nftBackend {
	if _, err := exec.LookPath("nft"); err != nil {
		return nil
	}

	return &nftBackend{
		appTable: "outway",
		entries:  make(map[string]nftEntry),
	}
}

func (n *nftBackend) Name() string { return "nftables" }

func (n *nftBackend) EnsurePolicy(ctx context.Context, iface string) error {
	zerolog.Ctx(ctx).Info().Str("iface", iface).Msg("ensure nft policy")

	// Validate interface name to mitigate command-injection risks
	if !isSafeIfaceName(iface) {
		return fmt.Errorf("%w: %q", ErrInvalidIface, iface)
	}

	// Create table and set if not exist
	cmd := exec.CommandContext(ctx, "nft", "add", "table", "inet", n.appTable) //nolint:gosec // nft is a system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Err(err).Str("out", string(out)).Msg("nft add table (may already exist)")
	}

	cmd = exec.CommandContext(ctx, "nft", "add", "set", "inet", n.appTable, ifaceSet(iface),
		"{ type ipv4_addr; flags timeout; } ") // #nosec G204 -- args validated; no shell; nft system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Err(err).Str("out", string(out)).Str("set", ifaceSet(iface)).Msg("nft add set IPv4 (may already exist)")
	}

	cmd = exec.CommandContext(ctx, "nft", "add", "set", "inet", n.appTable, ifaceSet6(iface),
		"{ type ipv6_addr; flags timeout; } ") // #nosec G204 -- args validated; no shell; nft system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Err(err).Str("out", string(out)).Str("set", ifaceSet6(iface)).Msg("nft add set IPv6 (may already exist)")
	}

	return nil
}

func (n *nftBackend) MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Validate inputs to satisfy gosec G204 and avoid unsafe args
	if !isSafeIfaceName(iface) {
		return fmt.Errorf("%w: %q", ErrInvalidIface, iface)
	}

	normalizedIP, ok := normalizeIP(ip)
	if !ok {
		return fmt.Errorf("%w: %q", ErrInvalidIP, ip)
	}

	if ttlSeconds < minTTLSeconds {
		ttlSeconds = minTTLSeconds
	}

	// Check if IP is already marked with longer TTL
	if existing, exists := n.entries[normalizedIP]; exists {
		if existing.expireAt.After(time.Now().Add(time.Duration(ttlSeconds) * time.Second)) {
			zerolog.Ctx(ctx).Debug().Str("ip", normalizedIP).Str("iface", iface).Msg("IP already marked with longer TTL, skipping")

			return nil
		}
	}

	zerolog.Ctx(ctx).Debug().Str("iface", iface).Str("ip", normalizedIP).Int("ttl", ttlSeconds).Msg("mark ip")

	var set string
	if isIPv6(normalizedIP) {
		set = ifaceSet6(iface)
	} else {
		set = ifaceSet(iface)
	}

	cmd := exec.CommandContext(ctx, "nft", "add", "element", "inet", n.appTable, set,
		fmt.Sprintf("{ %s timeout %ds }", normalizedIP, ttlSeconds)) // #nosec G204 -- args validated; no shell; nft system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Str("out", string(out)).
			Str("ip", normalizedIP).Str("set", set).Str("iface", iface).
			Msg("nft add element failed")

		return fmt.Errorf("failed to add IP %s to set %s: %w", normalizedIP, set, err)
	}

	zerolog.Ctx(ctx).Debug().Str("ip", normalizedIP).Str("set", set).Str("iface", iface).Int("ttl", ttlSeconds).Msg("nft add element success")
	// Track expiry; best-effort cleanup timer
	exp := time.Now().Add(time.Duration(ttlSeconds) * time.Second)

	n.entries[normalizedIP] = nftEntry{
		expireAt: exp,
		iface:    iface,
	}
	go n.expireAfter(ctx, normalizedIP, iface, exp)

	return nil
}

func (n *nftBackend) CleanupAll(ctx context.Context) error {
	zerolog.Ctx(ctx).Info().Str("table", n.appTable).Msg("cleanup nft table")

	n.mutex.Lock()
	n.entries = make(map[string]nftEntry) // Clear local tracking
	n.mutex.Unlock()

	cmd := exec.CommandContext(ctx, "nft", "delete", "table", "inet", n.appTable) //nolint:gosec // nft is a system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Err(err).Str("out", string(out)).Msg("nft delete table (may not exist)")
	}

	return nil
}

func (n *nftBackend) expireAfter(ctx context.Context, ip, iface string, expireAt time.Time) {
	t := time.NewTimer(time.Until(expireAt))
	<-t.C

	n.mutex.Lock()
	defer n.mutex.Unlock()

	// Check if entry still exists and hasn't been updated
	entry, exists := n.entries[ip]
	if !exists || !entry.expireAt.Equal(expireAt) {
		// Entry was updated or removed, don't delete
		return
	}

	// Determine correct set based on IP type and interface
	var set string
	if isIPv6(ip) {
		set = ifaceSet6(iface)
	} else {
		set = ifaceSet(iface)
	}

	cmd := exec.CommandContext(ctx, "nft", "delete", "element", "inet", n.appTable, set,
		fmt.Sprintf("{ %s }", ip)) // #nosec G204 -- args validated earlier; no shell; nft system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Err(err).Str("out", string(out)).Str("ip", ip).Str("set", set).Msg("nft delete element (may already be expired)")
	}

	delete(n.entries, ip)
}

func ifaceSet(iface string) string  { return iface + "_v4" }
func ifaceSet6(iface string) string { return iface + "_v6" }

// isSafeIfaceName verifies interface names to a conservative charset to avoid injection via args.
var ifaceNameRe = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,32}$`)

func isSafeIfaceName(iface string) bool {
	return ifaceNameRe.MatchString(iface)
}

// normalizeIP parses and returns canonical string representation without brackets.
func normalizeIP(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	// Strip brackets that sometimes wrap IPv6
	if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
		trimmed = strings.TrimPrefix(strings.TrimSuffix(trimmed, "]"), "[")
	}
	// Remove possible zone identifier (e.g., fe80::1%eth0)
	host := trimmed
	if i := strings.IndexByte(trimmed, '%'); i >= 0 {
		host = trimmed[:i]
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return "", false
	}

	return ip.String(), true
}

func isIPv6(ip string) bool {
	// More reliable IPv6 detection
	// Check for IPv6 indicators: colons, brackets, or IPv4-mapped format
	hasColon := false
	hasDot := false
	hasBracket := false

	for _, char := range ip {
		switch char {
		case ':':
			hasColon = true
		case '.':
			hasDot = true
		case '[', ']':
			hasBracket = true
		}
	}

	// IPv6 if:
	// 1. Has brackets (IPv6 with port)
	// 2. Has colons and no dots (pure IPv6)
	// 3. Has colons and dots but starts with ::ffff: (IPv4-mapped IPv6)
	return hasBracket || (hasColon && !hasDot) || (hasColon && hasDot && strings.HasPrefix(ip, "::ffff:"))
}
