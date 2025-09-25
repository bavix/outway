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

const (
	hashMultiplier   = 31
	markRangeBase    = 1000000
	maxMarkValue     = 2147483647
	minimumMarkValue = 1000
)

var errNoGatewayFound = errors.New("no gateway found for interface")

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
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Msg("nft add table (may already exist)")
	}

	// Create IPv4 set
	cmd = exec.CommandContext(ctx, "nft", "add", "set", "inet", n.appTable, ifaceSet(iface),
		"{ type ipv4_addr; flags timeout; } ") // #nosec G204 -- args validated; no shell; nft system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("set", ifaceSet(iface)).Msg("nft add set IPv4 (may already exist)")
	}

	// Create IPv6 set
	cmd = exec.CommandContext(ctx, "nft", "add", "set", "inet", n.appTable, ifaceSet6(iface),
		"{ type ipv6_addr; flags timeout; } ") // #nosec G204 -- args validated; no shell; nft system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("set", ifaceSet6(iface)).Msg("nft add set IPv6 (may already exist)")
	}

	// CRITICAL FIX: Create routing rules for OpenWrt firewall4
	if err := n.ensureRoutingRules(ctx, iface); err != nil {
		zerolog.Ctx(ctx).Err(err).Str("iface", iface).Msg("failed to create routing rules")

		return err
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
			zerolog.Ctx(ctx).Debug().IPAddr("ip", net.ParseIP(normalizedIP)).Str("iface", iface).Msg("IP already marked with longer TTL, skipping")

			return nil
		}
	}

	zerolog.Ctx(ctx).Debug().Str("iface", iface).IPAddr("ip", net.ParseIP(normalizedIP)).Int("ttl", ttlSeconds).Msg("mark ip")

	var set string
	if isIPv6(normalizedIP) {
		set = ifaceSet6(iface)
	} else {
		set = ifaceSet(iface)
	}

	cmd := exec.CommandContext(ctx, "nft", "add", "element", "inet", n.appTable, set,
		fmt.Sprintf("{ %s timeout %ds }", normalizedIP, ttlSeconds)) // #nosec G204 -- args validated; no shell; nft system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("out", out).
			IPAddr("ip", net.ParseIP(normalizedIP)).Str("set", set).Str("iface", iface).
			Msg("nft add element failed")

		return fmt.Errorf("failed to add IP %s to set %s: %w", normalizedIP, set, err)
	}

	zerolog.Ctx(ctx).Debug().
		IPAddr("ip", net.ParseIP(normalizedIP)).
		Str("set", set).
		Str("iface", iface).
		Int("ttl", ttlSeconds).
		Msg("nft add element success")
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

	ifaces := make([]string, 0, len(n.entries))
	for _, entry := range n.entries {
		// Collect unique interfaces
		found := false

		for _, iface := range ifaces {
			if iface == entry.iface {
				found = true

				break
			}
		}

		if !found {
			ifaces = append(ifaces, entry.iface)
		}
	}

	n.entries = make(map[string]nftEntry) // Clear local tracking
	n.mutex.Unlock()

	// Cleanup routing rules for each interface
	for _, iface := range ifaces {
		n.cleanupRoutingRules(ctx, iface)
	}

	// Cleanup outway mark chain
	cmd := exec.CommandContext(ctx, "nft", "delete", "chain", "inet", "mangle", "outway_mark")
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Msg("nft delete mark chain (may not exist)")
	}

	// Delete main table
	cmd = exec.CommandContext(ctx, "nft", "delete", "table", "inet", n.appTable) //nolint:gosec // nft is a system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Msg("nft delete table (may not exist)")
	}

	return nil
}

// cleanupRoutingRules removes policy routing rules for an interface..
func (n *nftBackend) cleanupRoutingRules(ctx context.Context, iface string) {
	mark := n.getMarkForIface(iface)
	tableName := "outway_" + iface

	// Remove ip rule
	cmd := exec.CommandContext(ctx, "ip", "rule", "del", "fwmark", mark, "table", tableName) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("mark", mark).Msg("ip rule del (may not exist)")
	}

	// Remove routing table
	cmd = exec.CommandContext(ctx, "ip", "route", "flush", "table", tableName) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("table", tableName).Msg("ip route flush table (may not exist)")
	}
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
		zerolog.Ctx(ctx).Debug().
			Bytes("out", out).
			IPAddr("ip", net.ParseIP(ip)).
			Str("set", set).
			Msg("nft delete element (may already be expired)")
	}

	delete(n.entries, ip)
}

func ifaceSet(iface string) string  { return iface + "_v4" }
func ifaceSet6(iface string) string { return iface + "_v6" }

// ensureRoutingRules creates nftables rules for packet marking and policy routing..
func (n *nftBackend) ensureRoutingRules(ctx context.Context, iface string) error {
	// Generate a unique mark for this interface
	mark := n.getMarkForIface(iface)

	// Create chain for packet marking (hook on mangle table)
	cmd := exec.CommandContext(ctx, "nft", "add", "table", "inet", "mangle")
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Msg("nft add mangle table (may already exist)")
	}

	// Create chain for marking packets destined to our IP sets
	cmd = exec.CommandContext(ctx, "nft", "add", "chain", "inet", "mangle", "outway_mark",
		"{ type filter hook forward priority mangle; policy accept; }")
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Msg("nft add mark chain (may already exist)")
	}

	// Add rule to mark IPv4 packets destined to IPs in our set
	cmd = exec.CommandContext(ctx, "nft", "add", "rule", "inet", "mangle", "outway_mark", //nolint:gosec
		"ip daddr @"+ifaceSet(iface)+" meta mark set "+mark)
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("out", out).Str("rule", "ipv4 mark").Msg("nft add IPv4 mark rule failed")

		return fmt.Errorf("failed to add IPv4 mark rule: %w", err)
	}

	// Add rule to mark IPv6 packets destined to IPs in our set
	cmd = exec.CommandContext(ctx, "nft", "add", "rule", "inet", "mangle", "outway_mark", //nolint:gosec
		"ip6 daddr @"+ifaceSet6(iface)+" meta mark set "+mark)
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("out", out).Str("rule", "ipv6 mark").Msg("nft add IPv6 mark rule failed")

		return fmt.Errorf("failed to add IPv6 mark rule: %w", err)
	}

	// Setup policy routing using ip rule and ip route
	return n.setupPolicyRouting(ctx, iface, mark)
}

// getMarkForIface generates a unique mark value for an interface..
func (n *nftBackend) getMarkForIface(iface string) string {
	// Simple hash of interface name to generate consistent mark
	hash := 0
	for _, c := range iface {
		hash = hash*hashMultiplier + int(c)
	}
	// Ensure mark is in valid range (1-2147483647) and avoid common values
	mark := (hash%markRangeBase + markRangeBase) % maxMarkValue
	if mark < minimumMarkValue {
		mark += minimumMarkValue // Avoid low mark values
	}

	return fmt.Sprintf("0x%x", mark)
}

// setupPolicyRouting configures ip rule and ip route for policy routing.
func (n *nftBackend) setupPolicyRouting(ctx context.Context, iface, mark string) error {
	tableName := "outway_" + iface

	// Add ip rule for marked packets
	cmd := exec.CommandContext(ctx, "ip", "rule", "add", "fwmark", mark, "table", tableName) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("mark", mark).Msg("ip rule add (may already exist)")
	}

	// Add default route via the interface
	cmd = exec.CommandContext(ctx, "ip", "route", "add", "default", "dev", iface, "table", tableName) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("out", out).Str("iface", iface).Msg("ip route add failed")
		// Try to get gateway for the interface
		if gw, err := n.getInterfaceGateway(ctx, iface); err == nil && gw != "" {
			cmd = exec.CommandContext(ctx, "ip", "route", "add", "default", "via", gw, "dev", iface, "table", tableName) //nolint:gosec
			if out, err := cmd.CombinedOutput(); err != nil {
				zerolog.Ctx(ctx).Err(err).Bytes("out", out).Str("iface", iface).Str("gw", gw).Msg("ip route add via gateway failed")

				return fmt.Errorf("failed to add route via gateway %s for interface %s: %w", gw, iface, err)
			}
		} else {
			return fmt.Errorf("failed to add route for interface %s: %w", iface, err)
		}
	}

	return nil
}

// getInterfaceGateway attempts to get the gateway for an interface.
func (n *nftBackend) getInterfaceGateway(ctx context.Context, iface string) (string, error) {
	// Try to get gateway from ip route
	cmd := exec.CommandContext(ctx, "ip", "route", "show", "dev", iface)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "default") && strings.Contains(line, "via") {
			// Parse "default via 192.168.1.1 dev eth0"
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "via" && i+1 < len(parts) {
					return parts[i+1], nil
				}
			}
		}
	}

	return "", fmt.Errorf("%w: %s", errNoGatewayFound, iface)
}

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
