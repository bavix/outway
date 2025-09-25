package firewall

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
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
	// Routing table ID range for outway (100-199).
	routingTableBase = 100
	routingTableMax  = 199
	maxRetryAttempts = 2
)

var errNoGatewayFound = errors.New("no gateway found for interface")

type nftBackend struct {
	mutex        sync.Mutex
	appTable     string
	entries      map[string]nftEntry // ip -> entry info
	routingTable int                 // Dynamic routing table ID
}

type nftEntry struct {
	expireAt time.Time
	iface    string
}

var (
	ErrInvalidIface = errors.New("invalid interface name")
	ErrInvalidIP    = errors.New("invalid IP address")
)

func newNFTBackend(ctx context.Context) *nftBackend {
	if _, err := exec.LookPath("nft"); err != nil {
		return nil
	}

	backend := &nftBackend{
		appTable:     "outway",
		entries:      make(map[string]nftEntry),
		routingTable: 0, // Will be set dynamically
	}

	// Find and reserve a free routing table
	backend.routingTable = backend.findFreeRoutingTable(ctx)

	return backend
}

func (n *nftBackend) Name() string { return "nftables" }

func (n *nftBackend) EnsurePolicy(ctx context.Context, iface string) error {
	zerolog.Ctx(ctx).Info().Str("iface", iface).Int("table", n.routingTable).Msg("ensure nft policy")

	// Validate interface name to mitigate command-injection risks
	if !isSafeIfaceName(iface) {
		return fmt.Errorf("%w: %q", ErrInvalidIface, iface)
	}

	// Clean up any existing routes/rules for our table before starting
	n.cleanupRoutingTable(ctx)

	// Setup nftables tables
	if err := n.setupNFTablesTables(ctx); err != nil {
		return err
	}

	// Create IP sets for the interface
	if err := n.createIPSets(ctx, iface); err != nil {
		return err
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

	// Validate inputs
	if err := n.validateMarkIPInputs(iface, ip); err != nil {
		return err
	}

	normalizedIP, _ := normalizeIP(ip)
	ttlSeconds = n.normalizeTTL(ttlSeconds)

	// Check if IP is already marked with longer TTL
	if n.isIPAlreadyMarked(normalizedIP, ttlSeconds) {
		zerolog.Ctx(ctx).Debug().IPAddr("ip", net.ParseIP(normalizedIP)).Str("iface", iface).Msg("IP already marked with longer TTL, skipping")

		return nil
	}

	// Add IP to nftables set
	set := n.getSetForIP(normalizedIP, iface)
	if err := n.addIPToSet(ctx, normalizedIP, set, ttlSeconds); err != nil {
		return err
	}

	// Track expiry
	n.trackIPExpiry(ctx, normalizedIP, iface, ttlSeconds)

	return nil
}

func (n *nftBackend) CleanupAll(ctx context.Context) error {
	zerolog.Ctx(ctx).Info().Str("table", n.appTable).Int("routing_table", n.routingTable).Msg("cleanup nft table")

	// Collect unique interfaces and clear entries
	ifaces := n.collectAndClearInterfaces()

	// Cleanup routing table completely
	n.cleanupRoutingTable(ctx)

	// Cleanup sets from mangle table for each interface
	n.cleanupInterfaceSets(ctx, ifaces)

	// Cleanup nftables chains and tables
	n.cleanupNFTablesChains(ctx)
	n.cleanupNFTablesTables(ctx)

	return nil
}

// setupNFTablesTables creates the required nftables tables.
func (n *nftBackend) setupNFTablesTables(ctx context.Context) error {
	// Create main table
	cmd := exec.CommandContext(ctx, "nft", "add", "table", "inet", n.appTable) //nolint:gosec // nft is a system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Msg("nft add table (may already exist)")
	}

	// Ensure mangle table exists (required for sets and rules)
	cmd = exec.CommandContext(ctx, "nft", "add", "table", "inet", "mangle")
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Msg("nft add mangle table (may already exist)")
	}

	return nil
}

// createIPSets creates IPv4 and IPv6 sets for the interface.
func (n *nftBackend) createIPSets(ctx context.Context, iface string) error {
	// Create IPv4 set
	if err := n.createIPSetWithRetry(ctx, ifaceSet(iface), "ipv4_addr", "IPv4"); err != nil {
		return err
	}

	// Create IPv6 set
	if err := n.createIPSetWithRetry(ctx, ifaceSet6(iface), "ipv6_addr", "IPv6"); err != nil {
		return err
	}

	return nil
}

// createIPSetWithRetry creates an IP set with retry logic.
func (n *nftBackend) createIPSetWithRetry(ctx context.Context, setName, setType, setDesc string) error {
	var (
		out []byte
		err error
	)

	for attempt := range 3 {
		cmd := exec.CommandContext(ctx, "nft", "add", "set", "inet", "mangle", setName,
			"{ type "+setType+"; flags timeout; }") // #nosec G204 -- args validated; no shell; nft system utility

		out, err = cmd.CombinedOutput()
		if err == nil {
			zerolog.Ctx(ctx).Info().Str("set", setName).Msg("nft " + setDesc + " set created successfully")

			break
		}

		if attempt < maxRetryAttempts {
			zerolog.Ctx(ctx).Debug().Int("attempt", attempt+1).Str("set", setName).Msg("nft add " + setDesc + " set retry")
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		} else {
			zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("set", setName).Msg("nft add set " + setDesc + " (may already exist)")
		}
	}

	return err
}

// validateMarkIPInputs validates the inputs for MarkIP.
func (n *nftBackend) validateMarkIPInputs(iface, ip string) error {
	if !isSafeIfaceName(iface) {
		return fmt.Errorf("%w: %q", ErrInvalidIface, iface)
	}

	if _, ok := normalizeIP(ip); !ok {
		return fmt.Errorf("%w: %q", ErrInvalidIP, ip)
	}

	return nil
}

// normalizeTTL ensures TTL is at least the minimum value.
func (n *nftBackend) normalizeTTL(ttlSeconds int) int {
	if ttlSeconds < minTTLSeconds {
		return minTTLSeconds
	}

	return ttlSeconds
}

// isIPAlreadyMarked checks if IP is already marked with longer TTL.
func (n *nftBackend) isIPAlreadyMarked(normalizedIP string, ttlSeconds int) bool {
	existing, exists := n.entries[normalizedIP]

	return exists && existing.expireAt.After(time.Now().Add(time.Duration(ttlSeconds)*time.Second))
}

// getSetForIP returns the appropriate set name for the IP type.
func (n *nftBackend) getSetForIP(normalizedIP, iface string) string {
	if isIPv6(normalizedIP) {
		return ifaceSet6(iface)
	}

	return ifaceSet(iface)
}

// addIPToSet adds an IP to the nftables set with retry logic.
func (n *nftBackend) addIPToSet(ctx context.Context, normalizedIP, set string, ttlSeconds int) error {
	zerolog.Ctx(ctx).Debug().Str("iface", set).IPAddr("ip", net.ParseIP(normalizedIP)).Int("ttl", ttlSeconds).Msg("mark ip")

	var (
		out []byte
		err error
	)

	for attempt := range 3 {
		cmd := exec.CommandContext(ctx, "nft", "add", "element", "inet", "mangle", set,
			fmt.Sprintf("{ %s timeout %ds }", normalizedIP, ttlSeconds)) // #nosec G204 -- args validated; no shell; nft system utility

		out, err = cmd.CombinedOutput()
		if err == nil {
			break
		}

		if attempt < maxRetryAttempts {
			zerolog.Ctx(ctx).Debug().Int("attempt", attempt+1).Msg("nft add element retry")
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	if err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("out", out).
			IPAddr("ip", net.ParseIP(normalizedIP)).Str("set", set).
			Msg("nft add element failed")

		return fmt.Errorf("failed to add IP %s to set %s: %w", normalizedIP, set, err)
	}

	zerolog.Ctx(ctx).Debug().
		IPAddr("ip", net.ParseIP(normalizedIP)).
		Str("set", set).
		Int("ttl", ttlSeconds).
		Msg("nft add element success")

	return nil
}

// trackIPExpiry tracks the IP expiry and starts cleanup timer.
func (n *nftBackend) trackIPExpiry(ctx context.Context, normalizedIP, iface string, ttlSeconds int) {
	exp := time.Now().Add(time.Duration(ttlSeconds) * time.Second)

	n.entries[normalizedIP] = nftEntry{
		expireAt: exp,
		iface:    iface,
	}
	go n.expireAfter(ctx, normalizedIP, iface, exp)
}

// collectAndClearInterfaces collects unique interfaces and clears entries.
func (n *nftBackend) collectAndClearInterfaces() []string {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	ifaces := make([]string, 0, len(n.entries))
	for _, entry := range n.entries {
		if !n.containsInterface(ifaces, entry.iface) {
			ifaces = append(ifaces, entry.iface)
		}
	}

	n.entries = make(map[string]nftEntry) // Clear local tracking

	return ifaces
}

// containsInterface checks if the interface is already in the list.
func (n *nftBackend) containsInterface(ifaces []string, target string) bool {
	for _, iface := range ifaces {
		if iface == target {
			return true
		}
	}

	return false
}

// cleanupInterfaceSets deletes IPv4 and IPv6 sets for each interface.
func (n *nftBackend) cleanupInterfaceSets(ctx context.Context, ifaces []string) {
	for _, iface := range ifaces {
		n.deleteIPSet(ctx, ifaceSet(iface), "IPv4")
		n.deleteIPSet(ctx, ifaceSet6(iface), "IPv6")
	}
}

// deleteIPSet deletes an IP set from nftables.
func (n *nftBackend) deleteIPSet(ctx context.Context, setName, setDesc string) {
	cmd := exec.CommandContext(ctx, "nft", "delete", "set", "inet", "mangle", setName)
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("set", setName).Msg("nft delete " + setDesc + " set (may not exist)")
	}
}

// cleanupNFTablesChains cleans up nftables chains.
func (n *nftBackend) cleanupNFTablesChains(ctx context.Context) {
	// Cleanup outway mark chain (delete rules first, then chain)
	cmd := exec.CommandContext(ctx, "nft", "flush", "chain", "inet", "mangle", "outway_mark")
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Msg("nft flush mark chain (may not exist)")
	}

	cmd = exec.CommandContext(ctx, "nft", "delete", "chain", "inet", "mangle", "outway_mark")
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Msg("nft delete mark chain (may not exist)")
	}
}

// cleanupNFTablesTables cleans up nftables tables.
func (n *nftBackend) cleanupNFTablesTables(ctx context.Context) {
	// Delete main table
	cmd := exec.CommandContext(ctx, "nft", "delete", "table", "inet", n.appTable) //nolint:gosec // nft is a system utility
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Msg("nft delete table (may not exist)")
	}
}

// findFreeRoutingTable finds an available routing table ID in the range 100-199.
func (n *nftBackend) findFreeRoutingTable(ctx context.Context) int {
	// Try to get list of existing routing tables
	cmd := exec.CommandContext(ctx, "ip", "route", "show", "table", "all")

	out, err := cmd.CombinedOutput()
	if err != nil {
		// If we can't get the list, use the base table
		return routingTableBase
	}

	// Parse existing table IDs from output
	usedTables := make(map[int]bool)

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "table") {
			// Extract table ID from lines like "default via 192.168.1.1 dev eth0 table 100"
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "table" && i+1 < len(parts) {
					if tableID, err := strconv.Atoi(parts[i+1]); err == nil {
						usedTables[tableID] = true
					}
				}
			}
		}
	}

	// Find first available table ID in our range
	for tableID := routingTableBase; tableID <= routingTableMax; tableID++ {
		if !usedTables[tableID] {
			return tableID
		}
	}

	// If all tables are used, use the base table (will overwrite existing)
	return routingTableBase
}

// cleanupRoutingTable removes all routes and rules for our routing table.
func (n *nftBackend) cleanupRoutingTable(ctx context.Context) {
	if n.routingTable == 0 {
		return
	}

	tableID := strconv.Itoa(n.routingTable)
	n.flushRoutingTable(ctx, tableID)
	n.removeRoutingRules(ctx, tableID)
}

// flushRoutingTable removes all routes from the specified table.
func (n *nftBackend) flushRoutingTable(ctx context.Context, tableID string) {
	cmd := exec.CommandContext(ctx, "ip", "route", "flush", "table", tableID)
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("table", tableID).Msg("ip route flush table (may not exist)")
	}
}

// removeRoutingRules removes all rules pointing to our table.
func (n *nftBackend) removeRoutingRules(ctx context.Context, tableID string) {
	cmd := exec.CommandContext(ctx, "ip", "rule", "show")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "table "+tableID) {
			n.deleteRule(ctx, line, tableID)
		}
	}
}

// deleteRule deletes a specific routing rule.
func (n *nftBackend) deleteRule(ctx context.Context, line, tableID string) {
	parts := strings.Fields(line)

	const minParts = 2
	if len(parts) < minParts {
		return
	}

	// Reconstruct rule deletion command
	ruleParts := []string{"ip", "rule", "del"}

	for _, part := range parts {
		if part != "table" && part != tableID {
			ruleParts = append(ruleParts, part)
		}
	}

	cmd := exec.CommandContext(ctx, ruleParts[0], ruleParts[1:]...) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("rule", line).Msg("ip rule del (may not exist)")
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

	cmd := exec.CommandContext(ctx, "nft", "delete", "element", "inet", "mangle", set,
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
	mark := n.getMarkForIface(iface)

	// Setup nftables infrastructure
	if err := n.setupNFTablesInfrastructure(ctx); err != nil {
		return err
	}

	// Create IPv4 marking rule
	if err := n.createIPv4MarkingRule(ctx, iface, mark); err != nil {
		return err
	}

	// Create IPv6 marking rule
	if err := n.createIPv6MarkingRule(ctx, iface, mark); err != nil {
		return err
	}

	// Setup policy routing using ip rule and ip route
	return n.setupPolicyRouting(ctx, iface, mark)
}

// setupNFTablesInfrastructure creates the basic nftables table and chain.
func (n *nftBackend) setupNFTablesInfrastructure(ctx context.Context) error {
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

	return nil
}

// createIPv4MarkingRule creates the IPv4 packet marking rule.
func (n *nftBackend) createIPv4MarkingRule(ctx context.Context, iface, mark string) error {
	// Verify IPv4 set exists before creating rules
	cmd := exec.CommandContext(ctx, "nft", "list", "set", "inet", "mangle", ifaceSet(iface)) //nolint:gosec // iface validated
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("out", out).Str("set", ifaceSet(iface)).Msg("IPv4 set does not exist")

		return fmt.Errorf("IPv4 set %s does not exist: %w", ifaceSet(iface), err)
	}

	// Add rule to mark IPv4 packets destined to IPs in our set (with retry)
	return n.addMarkingRuleWithRetry(ctx, "ip", ifaceSet(iface), mark, "ipv4 mark")
}

// createIPv6MarkingRule creates the IPv6 packet marking rule.
func (n *nftBackend) createIPv6MarkingRule(ctx context.Context, iface, mark string) error {
	// Verify IPv6 set exists before creating rules
	cmd := exec.CommandContext(ctx, "nft", "list", "set", "inet", "mangle", ifaceSet6(iface)) //nolint:gosec // iface validated
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("out", out).Str("set", ifaceSet6(iface)).Msg("IPv6 set does not exist")

		return fmt.Errorf("IPv6 set %s does not exist: %w", ifaceSet6(iface), err)
	}

	// Add rule to mark IPv6 packets destined to IPs in our set (with retry)
	return n.addMarkingRuleWithRetry(ctx, "ip6", ifaceSet6(iface), mark, "ipv6 mark")
}

// addMarkingRuleWithRetry adds a marking rule with retry logic.
func (n *nftBackend) addMarkingRuleWithRetry(ctx context.Context, ipType, set, mark, ruleName string) error {
	var (
		out []byte
		err error
	)

	for attempt := range 3 {
		cmd := exec.CommandContext(ctx, "nft", "add", "rule", "inet", "mangle", "outway_mark", //nolint:gosec
			ipType+" daddr @"+set+" meta mark set "+mark)

		out, err = cmd.CombinedOutput()
		if err == nil {
			break
		}

		if attempt < maxRetryAttempts {
			zerolog.Ctx(ctx).Debug().Int("attempt", attempt+1).Str("rule", ruleName).Msg("nft add rule retry")
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	if err != nil {
		zerolog.Ctx(ctx).Err(err).Bytes("out", out).Str("rule", ruleName).Msg("nft add mark rule failed")

		return fmt.Errorf("failed to add %s rule: %w", ruleName, err)
	}

	return nil
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
	// Use dynamic table ID
	tableName := strconv.Itoa(n.routingTable)
	zerolog.Ctx(ctx).Info().Str("iface", iface).Str("mark", mark).Str("table", tableName).Msg("setting up policy routing")

	// Add ip rule for marked packets
	cmd := exec.CommandContext(ctx, "ip", "rule", "add", "fwmark", mark, "table", tableName) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("mark", mark).Msg("ip rule add (may already exist)")
	}

	// For tun interfaces, try to get gateway first, then fall back to dev-only route
	var routeAdded bool

	// Try to get gateway for the interface
	if gw, err := n.getInterfaceGateway(ctx, iface); err == nil && gw != "" {
		cmd = exec.CommandContext(ctx, "ip", "route", "add", "default", "via", gw, "dev", iface, "table", tableName) //nolint:gosec
		if out, err := cmd.CombinedOutput(); err != nil {
			zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("iface", iface).Str("gw", gw).
				Msg("ip route add via gateway failed, trying dev-only route")
		} else {
			routeAdded = true
		}
	}

	// If gateway route failed or no gateway found, try dev-only route
	if !routeAdded {
		cmd = exec.CommandContext(ctx, "ip", "route", "add", "default", "dev", iface, "table", tableName) //nolint:gosec
		if out, err := cmd.CombinedOutput(); err != nil {
			zerolog.Ctx(ctx).Err(err).Bytes("out", out).Str("iface", iface).Msg("ip route add failed")

			return fmt.Errorf("failed to add route for interface %s: %w", iface, err)
		}
	}

	return nil
}

// getInterfaceGateway attempts to get the gateway for an interface.
func (n *nftBackend) getInterfaceGateway(ctx context.Context, iface string) (string, error) {
	// Try to get gateway from route commands
	if gateway := n.getGatewayFromRoutes(ctx, iface); gateway != "" {
		return gateway, nil
	}

	// For tun/tap interfaces, try to get the peer IP as gateway
	if n.isTunnelInterface(iface) {
		if gateway := n.getGatewayFromPeer(ctx, iface); gateway != "" {
			return gateway, nil
		}
	}

	zerolog.Ctx(ctx).Debug().Str("iface", iface).Msg("no gateway found")

	return "", fmt.Errorf("%w: %s", errNoGatewayFound, iface)
}

// getGatewayFromRoutes tries to find gateway from ip route commands.
func (n *nftBackend) getGatewayFromRoutes(ctx context.Context, iface string) string {
	// Try interface-specific route first
	if gateway := n.parseRouteOutput(ctx, "ip", "route", "show", "dev", iface); gateway != "" {
		return gateway
	}

	// Fall back to global routes
	zerolog.Ctx(ctx).Debug().Str("iface", iface).Msg("ip route show dev failed, trying global routes")

	return n.parseRouteOutput(ctx, "ip", "route", "show", "default")
}

// parseRouteOutput parses the output of ip route commands to find gateway.
func (n *nftBackend) parseRouteOutput(ctx context.Context, cmd string, args ...string) string {
	command := exec.CommandContext(ctx, cmd, args...)

	out, err := command.CombinedOutput()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if gateway := n.extractGatewayFromLine(line); gateway != "" {
			zerolog.Ctx(ctx).Debug().Str("gateway", gateway).Msg("found gateway")

			return gateway
		}
	}

	return ""
}

// extractGatewayFromLine extracts gateway from a route line.
func (n *nftBackend) extractGatewayFromLine(line string) string {
	if !strings.Contains(line, "default") || !strings.Contains(line, "via") {
		return ""
	}

	parts := strings.Fields(line)
	for i, part := range parts {
		if part == "via" && i+1 < len(parts) {
			return parts[i+1]
		}
	}

	return ""
}

// isTunnelInterface checks if the interface is a tunnel interface.
func (n *nftBackend) isTunnelInterface(iface string) bool {
	return strings.HasPrefix(iface, "tun") || strings.HasPrefix(iface, "tap")
}

// getGatewayFromPeer tries to get gateway from peer IP for tunnel interfaces.
func (n *nftBackend) getGatewayFromPeer(ctx context.Context, iface string) string {
	cmd := exec.CommandContext(ctx, "ip", "addr", "show", iface)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if gateway := n.extractPeerFromLine(line); gateway != "" {
			zerolog.Ctx(ctx).Debug().Str("iface", iface).Str("peer", gateway).Msg("using peer IP as gateway")

			return gateway
		}
	}

	return ""
}

// extractPeerFromLine extracts peer IP from interface address line.
func (n *nftBackend) extractPeerFromLine(line string) string {
	if !strings.Contains(line, "peer") {
		return ""
	}

	parts := strings.Fields(line)
	for i, part := range parts {
		if part == "peer" && i+1 < len(parts) {
			peerIP := parts[i+1]
			// Remove any trailing scope info
			if idx := strings.Index(peerIP, "/"); idx != -1 {
				peerIP = peerIP[:idx]
			}

			return peerIP
		}
	}

	return ""
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
