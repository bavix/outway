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
	// Routing table ID range for outway (100-199).
	routingTableBase = 100
	routingTableMax  = 199
	maxRetryAttempts = 2
	tableKeyword     = "table"
	ipByteRange      = 256 // For IP address calculation
	markerOffset1    = 100 // Offset for fallback marker calculation
	markerOffset2    = 200 // Offset for fallback marker calculation
)

var (
	ErrInvalidIface = errors.New("invalid interface name")
	ErrInvalidIP    = errors.New("invalid IP address")
)

type routeBackend struct {
	mutex        sync.Mutex
	appTable     string
	entries      map[string]routeEntry // ip -> entry info
	routingTable int                   // Dynamic routing table ID
}

type routeEntry struct {
	expireAt time.Time
	iface    string
}

func newRouteBackend(ctx context.Context) *routeBackend {
	backend := &routeBackend{
		appTable:     "outway",
		entries:      make(map[string]routeEntry),
		routingTable: 0, // Will be set dynamically
	}

	// Find and reserve a free routing table
	backend.routingTable = backend.findFreeRoutingTable(ctx)

	return backend
}

func (r *routeBackend) Name() string { return "route" }

func (r *routeBackend) EnsurePolicy(ctx context.Context, iface string) error {
	zerolog.Ctx(ctx).Info().Str("iface", iface).Int("table", r.routingTable).Msg("ensure routing policy")

	// Validate interface name to mitigate command-injection risks
	if !isSafeIfaceName(iface) {
		return fmt.Errorf("%w: %q", ErrInvalidIface, iface)
	}

	// Clean up any existing outway tables before starting
	r.cleanupOldOutwayTables(ctx)

	// Setup basic routing table
	if err := r.setupRoutingTable(ctx, iface); err != nil {
		zerolog.Ctx(ctx).Err(err).Str("iface", iface).Msg("failed to setup routing table")

		return err
	}

	return nil
}

func (r *routeBackend) MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Validate inputs
	if err := r.validateMarkIPInputs(iface, ip); err != nil {
		return err
	}

	normalizedIP, _ := normalizeIP(ip)
	ttlSeconds = r.normalizeTTL(ttlSeconds)

	// Check if IP is already marked with longer TTL
	if r.isIPAlreadyMarked(normalizedIP, ttlSeconds) {
		zerolog.Ctx(ctx).Debug().IPAddr("ip", net.ParseIP(normalizedIP)).Str("iface", iface).Msg("IP already marked with longer TTL, skipping")

		return nil
	}

	// Create route for this specific IP through VPN interface
	if err := r.createRouteForIP(ctx, normalizedIP, iface); err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).IPAddr("ip", net.ParseIP(normalizedIP)).Str("iface", iface).Msg("failed to create route for IP")

		return err
	}

	// Track expiry
	r.trackIPExpiry(ctx, normalizedIP, iface, ttlSeconds)

	return nil
}

func (r *routeBackend) CleanupAll(ctx context.Context) error {
	zerolog.Ctx(ctx).Info().Str("table", r.appTable).Int("routing_table", r.routingTable).Msg("cleanup routing table")

	// Clear local entries
	r.clearLocalEntries(ctx)

	// Cleanup routing table completely
	r.cleanupRoutingTable(ctx)

	return nil
}

// getMarkerIP generates a unique marker IP for the table ID.
func (r *routeBackend) getMarkerIP(ctx context.Context, tableID string) string {
	// Try to find a free marker IP in different ranges
	markerRanges := []string{
		"169.254",    // link-local
		"192.0.2",    // TEST-NET-1 (RFC 3330)
		"198.51.100", // TEST-NET-2 (RFC 3330)
		"203.0.113",  // TEST-NET-3 (RFC 3330)
	}

	id, _ := strconv.Atoi(tableID)
	x := (id / ipByteRange) % ipByteRange
	y := id % ipByteRange

	// Try each range until we find a free one
	for _, base := range markerRanges {
		markerIP := fmt.Sprintf("%s.%d.%d", base, x, y)
		if r.isMarkerIPFree(ctx, markerIP) {
			return markerIP
		}
	}

	// Fallback to link-local with different calculation
	return fmt.Sprintf("169.254.%d.%d", (id+markerOffset1)%ipByteRange, (id+markerOffset2)%ipByteRange)
}

// isMarkerIPFree checks if a marker IP is not used by other applications.
func (r *routeBackend) isMarkerIPFree(ctx context.Context, markerIP string) bool {
	// Check if this IP is already used in any routing table
	cmd := exec.CommandContext(ctx, "ip", "route", "show", "table", "all")

	out, err := cmd.CombinedOutput()
	if err != nil {
		// If we can't check, assume it's free
		return true
	}

	// Check if the marker IP is already in use
	return !strings.Contains(string(out), markerIP+"/32")
}

// hasOutwayMarker checks if a table contains any outway marker route.
func (r *routeBackend) hasOutwayMarker(output, tableIDStr string) bool {
	// Check all possible marker ranges
	markerRanges := []string{
		"169.254",    // link-local
		"192.0.2",    // TEST-NET-1
		"198.51.100", // TEST-NET-2
		"203.0.113",  // TEST-NET-3
	}

	id, _ := strconv.Atoi(tableIDStr)
	x := (id / ipByteRange) % ipByteRange
	y := id % ipByteRange

	// Check each range
	for _, base := range markerRanges {
		markerIP := fmt.Sprintf("%s.%d.%d/32", base, x, y)
		if strings.Contains(output, markerIP) && strings.Contains(output, "dev lo") {
			return true
		}
	}

	// Check fallback range
	fallbackIP := fmt.Sprintf("169.254.%d.%d/32", (id+markerOffset1)%ipByteRange, (id+markerOffset2)%ipByteRange)

	return strings.Contains(output, fallbackIP) && strings.Contains(output, "dev lo")
}

// cleanupOldOutwayTables finds and cleans up old outway tables.
func (r *routeBackend) cleanupOldOutwayTables(ctx context.Context) {
	// Find all routing tables
	cmd := exec.CommandContext(ctx, "ip", "route", "show", "table", "all")

	out, err := cmd.CombinedOutput()
	if err != nil {
		zerolog.Ctx(ctx).Debug().Err(err).Msg("failed to list routing tables")

		return
	}

	// Parse table IDs and check for outway marker
	lines := strings.Split(string(out), "\n")
	usedTables := make(map[int]bool)

	for _, line := range lines {
		if strings.Contains(line, "table") {
			// Extract table ID from lines like "default via 192.168.1.1 dev eth0 table 100"
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == tableKeyword && i+1 < len(parts) {
					if tableID, err := strconv.Atoi(parts[i+1]); err == nil {
						usedTables[tableID] = true
					}
				}
			}
		}
	}

	// Check each table for outway marker
	for tableID := range usedTables {
		if r.isOutwayTable(ctx, tableID) {
			zerolog.Ctx(ctx).Info().Int("table", tableID).Msg("found old outway table, cleaning up")
			tableIDStr := strconv.Itoa(tableID)
			r.flushRoutingTable(ctx, tableIDStr)
			r.removeRoutingRules(ctx, tableIDStr)
		}
	}
}

// isOutwayTable checks if a table contains the outway marker route.
func (r *routeBackend) isOutwayTable(ctx context.Context, tableID int) bool {
	cmd := exec.CommandContext(ctx, "ip", "route", "show", "table", strconv.Itoa(tableID)) //nolint:gosec // tableID is validated integer

	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Check for outway marker route in any of the marker ranges
	tableIDStr := strconv.Itoa(tableID)

	return r.hasOutwayMarker(string(out), tableIDStr)
}

// Helper functions

func (r *routeBackend) clearLocalEntries(ctx context.Context) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.entries = make(map[string]routeEntry)

	zerolog.Ctx(ctx).Debug().Msg("local entries cleared")
}

func (r *routeBackend) setupRoutingTable(ctx context.Context, iface string) error {
	tableID := strconv.Itoa(r.routingTable)

	// Create default route through the interface
	cmd := exec.CommandContext(ctx, "ip", "route", "add", "default", "dev", iface, "table", tableID) //nolint:gosec // iface validated
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("iface", iface).Str("table", tableID).Msg("ip route add default (may already exist)")
	}

	// Add unique marker route to identify outway tables
	markerIP := r.getMarkerIP(ctx, tableID)

	cmd = exec.CommandContext(ctx, "ip", "route", "add", markerIP+"/32", "dev", "lo", "table", tableID) //nolint:gosec // marker route
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("table", tableID).Str("marker", markerIP).Msg("ip route add marker (may already exist)")
	}

	zerolog.Ctx(ctx).Info().Str("iface", iface).Str("table", tableID).Msg("routing table setup complete")

	return nil
}

func (r *routeBackend) validateMarkIPInputs(iface, ip string) error {
	if !isSafeIfaceName(iface) {
		return fmt.Errorf("%w: %q", ErrInvalidIface, iface)
	}

	if _, ok := normalizeIP(ip); !ok {
		return fmt.Errorf("%w: %q", ErrInvalidIP, ip)
	}

	return nil
}

func (r *routeBackend) normalizeTTL(ttlSeconds int) int {
	if ttlSeconds < minTTLSeconds {
		return minTTLSeconds
	}

	return ttlSeconds
}

func (r *routeBackend) isIPAlreadyMarked(normalizedIP string, ttlSeconds int) bool {
	existing, exists := r.entries[normalizedIP]

	return exists && existing.expireAt.After(time.Now().Add(time.Duration(ttlSeconds)*time.Second))
}

func (r *routeBackend) trackIPExpiry(ctx context.Context, normalizedIP, iface string, ttlSeconds int) {
	exp := time.Now().Add(time.Duration(ttlSeconds) * time.Second)

	r.entries[normalizedIP] = routeEntry{
		expireAt: exp,
		iface:    iface,
	}
	go r.expireAfter(ctx, normalizedIP, iface, exp)
}

func (r *routeBackend) createRouteForIP(ctx context.Context, ip, iface string) error {
	// Create route: ip route add <ip>/32 dev <iface> table <routing_table>
	tableID := strconv.Itoa(r.routingTable)
	cmd := exec.CommandContext(ctx, "ip", "route", "add", ip+"/32", "dev", iface, "table", tableID) //nolint:gosec // ip and iface validated

	if out, err := cmd.CombinedOutput(); err != nil {
		// Check if route already exists (error 2 = file exists)
		if !strings.Contains(string(out), "File exists") {
			zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("ip", ip).Str("iface", iface).
				Str("table", tableID).Msg("ip route add (may already exist)")

			return fmt.Errorf("failed to add route for %s: %w", ip, err)
		}

		zerolog.Ctx(ctx).Debug().Str("ip", ip).Str("iface", iface).Str("table", tableID).Msg("route already exists")
	} else {
		zerolog.Ctx(ctx).Debug().Str("ip", ip).Str("iface", iface).Str("table", tableID).Msg("route created successfully")
	}

	return nil
}

func (r *routeBackend) removeRouteForIP(ctx context.Context, ip, iface string) {
	tableID := strconv.Itoa(r.routingTable)
	cmd := exec.CommandContext(ctx, "ip", "route", "del", ip+"/32", "dev", iface, "table", tableID) //nolint:gosec // ip and iface validated

	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("ip", ip).Str("iface", iface).Str("table", tableID).Msg("ip route del (may not exist)")
	} else {
		zerolog.Ctx(ctx).Debug().Str("ip", ip).Str("iface", iface).Str("table", tableID).Msg("route removed successfully")
	}
}

func (r *routeBackend) expireAfter(ctx context.Context, ip, iface string, expireAt time.Time) {
	t := time.NewTimer(time.Until(expireAt))
	<-t.C

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if entry still exists and hasn't been updated
	entry, exists := r.entries[ip]
	if !exists || !entry.expireAt.Equal(expireAt) {
		// Entry was updated or removed, don't delete
		return
	}

	// Remove route for this IP
	r.removeRouteForIP(ctx, ip, iface)

	delete(r.entries, ip)
}

// findFreeRoutingTable finds an available routing table ID in the range 100-199.
func (r *routeBackend) findFreeRoutingTable(ctx context.Context) int {
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
				if part == tableKeyword && i+1 < len(parts) {
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
func (r *routeBackend) cleanupRoutingTable(ctx context.Context) {
	if r.routingTable == 0 {
		return
	}

	tableID := strconv.Itoa(r.routingTable)
	r.flushRoutingTable(ctx, tableID)
	r.removeRoutingRules(ctx, tableID)
}

// flushRoutingTable removes all routes from the specified table.
func (r *routeBackend) flushRoutingTable(ctx context.Context, tableID string) {
	cmd := exec.CommandContext(ctx, "ip", "route", "flush", "table", tableID)
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("table", tableID).Msg("ip route flush table (may not exist)")
	}
}

// removeRoutingRules removes all rules pointing to our table.
func (r *routeBackend) removeRoutingRules(ctx context.Context, tableID string) {
	cmd := exec.CommandContext(ctx, "ip", "rule", "show")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "table "+tableID) {
			r.deleteRule(ctx, line, tableID)
		}
	}
}

// deleteRule deletes a specific routing rule.
func (r *routeBackend) deleteRule(ctx context.Context, line, tableID string) {
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
