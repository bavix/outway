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
	// Routing table ID range for outway (30000-30999).
	RoutingTableBase = 30000
	RoutingTableMax  = 30999
	Fw4TableName     = "fw4"
	maxRetryAttempts = 2
	tableKeyword     = "table"
	ipByteRange      = 256 // For IP address calculation
	markerOffset1    = 100 // Offset for fallback marker calculation
	markerOffset2    = 200 // Offset for fallback marker calculation

	// Outway marker constants.
	OutwayMarkerProto  = "outway"
	OutwayMarkerMetric = 999
	// Marker IP pool for outway tables (link-local range).
	MarkerIPPoolStart = "169.254.0.1"
	MarkerIPPoolEnd   = "169.254.0.100"

	// CRITICAL: System constants for validation.
	minSystemTableID     = 1000 // Minimum table ID to avoid system tables
	minTableRange        = 100  // Minimum range for table allocation
	systemCommandTimeout = 30   // Timeout for system commands in seconds
	tableSamplingStep    = 20   // Step for table sampling optimization

	// IP address bit shift constants.
	ipByteShift1 = 24 // First byte shift
	ipByteShift2 = 16 // Second byte shift
	ipByteShift3 = 8  // Third byte shift
)

var (
	ErrInvalidIface = errors.New("invalid interface name")
	ErrInvalidIP    = errors.New("invalid IP address")

	// CRITICAL: Static errors for better error handling.
	ErrNoTunnelsProvided      = errors.New("no tunnels provided")
	ErrEmptyTunnelName        = errors.New("empty tunnel name provided")
	ErrInvalidTunnelName      = errors.New("invalid tunnel name")
	ErrDuplicateTunnelName    = errors.New("duplicate tunnel name")
	ErrFwmarkRangeExhausted   = errors.New("fwmark range exhausted")
	ErrPriorityRangeExhausted = errors.New("priority range exhausted")
	ErrFwmarkRangeOverflow    = errors.New("fwmark range would overflow")
	ErrPriorityRangeOverflow  = errors.New("priority range would overflow")
	ErrTunnelNotFound         = errors.New("tunnel not found")
	ErrTunnelInitFailed       = errors.New("failed to initialize tunnel")
	ErrCleanupErrors          = errors.New("cleanup errors")
	ErrNoAvailableTableIDs    = errors.New("no available table IDs")
	ErrTableIDOutOfRange      = errors.New("table ID is outside allowed range")
	ErrInvalidMarkerIPPool    = errors.New("invalid marker IP pool range")
	ErrNoFreeMarkerIP         = errors.New("no free marker IP found in pool")
)

type routeBackend struct {
	mutex        sync.Mutex
	appTable     string
	entries      map[string]routeEntry // ip -> entry info
	tunnels      map[string]TunnelInfo // iface -> tunnel info
	nextTableID  int                   // Next available table ID
	nextFwMark   int                   // Next available fwmark
	nextPriority int                   // Next available priority
	nftTable     string                // Cached nftables table name
}

type routeEntry struct {
	expireAt time.Time
	iface    string
}

func NewRouteBackend() (*routeBackend, error) {
	backend := &routeBackend{
		appTable:     "outway",
		entries:      make(map[string]routeEntry),
		tunnels:      make(map[string]TunnelInfo),
		nextTableID:  RoutingTableBase,
		nextFwMark:   RoutingTableBase,
		nextPriority: RoutingTableBase,
		nftTable:     "", // Will be detected on first use
	}

	return backend, nil
}

func (r *routeBackend) Name() string { return "route" }

// InitializeTunnels initializes tunnel interfaces and returns their information.
func (r *routeBackend) InitializeTunnels(ctx context.Context, tunnels []string) ([]TunnelInfo, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if err := r.validateTunnelList(tunnels); err != nil {
		return nil, err
	}

	log := zerolog.Ctx(ctx)
	tunnelInfos := make([]TunnelInfo, 0, len(tunnels))

	if err := r.cleanupOldOutwayTables(ctx); err != nil {
		log.Warn().Err(err).Msg("failed to cleanup old outway tables, continuing anyway")
	}

	for _, tunnel := range tunnels {
		// Check if tunnel already exists
		if info, exists := r.tunnels[tunnel]; exists {
			tunnelInfos = append(tunnelInfos, info)

			continue
		}

		tableID, err := r.findOrCreateTableForTunnel(ctx, tunnel)
		if err != nil {
			return nil, fmt.Errorf("failed to find or create table for tunnel %s: %w", tunnel, err)
		}

		fwMark, priority := r.allocateNextMarkAndPriority()

		// Create tunnel info
		info := TunnelInfo{
			Name:     tunnel,
			TableID:  tableID,
			FwMark:   fwMark,
			Priority: priority,
		}

		// Find available marker IP
		markerIP, err := r.findAvailableMarkerIP(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to find available marker IP: %w", err)
		}

		// Setup routing table and rules
		r.setupTunnelTable(ctx, info, markerIP)

		// Store tunnel info
		r.tunnels[tunnel] = info
		tunnelInfos = append(tunnelInfos, info)

		log.Info().
			Str("tunnel", tunnel).
			Int("table", tableID).
			Int("fwmark", fwMark).
			Int("priority", priority).
			Msg("tunnel initialized")
	}

	return tunnelInfos, nil
}

// GetTunnelInfo returns information about a specific tunnel.
func (r *routeBackend) GetTunnelInfo(ctx context.Context, iface string) (*TunnelInfo, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	info, exists := r.tunnels[iface]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTunnelNotFound, iface)
	}

	return &info, nil
}

// FlushRuntime flushes runtime data from tables but keeps the table structure.
func (r *routeBackend) FlushRuntime(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log := zerolog.Ctx(ctx)
	log.Info().Msg("flushing runtime data from outway tables")

	for _, tunnel := range r.tunnels {
		// Flush routes from table but keep the table structure
		r.flushTableRoutes(ctx, tunnel.TableID)

		// Flush nft sets for this tunnel
		r.flushNftSets(ctx, tunnel.Name)
	}

	// Clear local entries
	r.entries = make(map[string]routeEntry)

	return nil
}

func (r *routeBackend) EnsurePolicy(ctx context.Context, iface string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Validate interface name to mitigate command-injection risks
	if !IsSafeIfaceName(iface) {
		return fmt.Errorf("%w: %q", ErrInvalidIface, iface)
	}

	// Check if tunnel already exists
	if _, exists := r.tunnels[iface]; exists {
		zerolog.Ctx(ctx).Info().Str("iface", iface).Msg("tunnel already configured")
	}

	// Initialize single tunnel
	tunnelInfos, err := r.InitializeTunnels(ctx, []string{iface})
	if err != nil {
		return err
	}

	if len(tunnelInfos) == 0 {
		return fmt.Errorf("%w: %s", ErrTunnelInitFailed, iface)
	}

	zerolog.Ctx(ctx).Info().
		Str("iface", iface).
		Int("table", tunnelInfos[0].TableID).
		Int("fwmark", tunnelInfos[0].FwMark).
		Msg("tunnel policy ensured")

	return nil
}

func (r *routeBackend) MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Validate inputs
	if err := r.validateMarkIPInputs(iface, ip); err != nil {
		return err
	}

	// Check if tunnel exists
	tunnel, exists := r.tunnels[iface]
	if !exists {
		return fmt.Errorf("%w: %s", ErrTunnelNotFound, iface)
	}

	normalizedIP, _ := NormalizeIP(ip)
	ttlSeconds = r.normalizeTTL(ttlSeconds)

	// Check if IP is already marked with longer TTL
	if r.isIPAlreadyMarked(normalizedIP, ttlSeconds) {
		zerolog.Ctx(ctx).Debug().IPAddr("ip", net.ParseIP(normalizedIP)).Str("iface", iface).Msg("IP already marked with longer TTL, skipping")
	}

	// Create route for this specific IP through VPN interface
	if err := r.createRouteForIPInTable(ctx, normalizedIP, iface, tunnel.TableID); err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).IPAddr("ip", net.ParseIP(normalizedIP)).Str("iface", iface).Msg("failed to create route for IP")

		return err
	}

	// Add IP to nft set for fwmark matching
	r.addIPToNftSet(ctx, normalizedIP, iface, ttlSeconds)

	// Track expiry
	r.trackIPExpiry(ctx, normalizedIP, iface, ttlSeconds)

	return nil
}

func (r *routeBackend) CleanupAll(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log := zerolog.Ctx(ctx)
	log.Info().Msg("cleanup all outway tables and rules")

	// Clear local entries
	r.entries = make(map[string]routeEntry)

	// Cleanup all tunnel tables
	for _, tunnel := range r.tunnels {
		tableID := strconv.Itoa(tunnel.TableID)

		// Flush table routes
		r.flushTableRoutes(ctx, tunnel.TableID)

		// Remove routing rules
		r.removeRoutingRulesForTable(ctx, tableID)

		// Remove nft sets
		r.removeNftSets(ctx, tunnel.Name)

		log.Info().
			Str("tunnel", tunnel.Name).
			Int("table", tunnel.TableID).
			Msg("tunnel cleaned up")
	}

	// Clear tunnels map
	r.tunnels = make(map[string]TunnelInfo)

	return nil
}

// validateTunnelList validates the list of tunnel names.
func (r *routeBackend) validateTunnelList(tunnels []string) error {
	if len(tunnels) == 0 {
		return ErrNoTunnelsProvided
	}

	seen := make(map[string]bool)

	for _, tunnel := range tunnels {
		if tunnel == "" {
			return ErrEmptyTunnelName
		}

		if !IsSafeIfaceName(tunnel) {
			return fmt.Errorf("%w: %s", ErrInvalidTunnelName, tunnel)
		}

		if seen[tunnel] {
			return fmt.Errorf("%w: %s", ErrDuplicateTunnelName, tunnel)
		}

		seen[tunnel] = true
	}

	return nil
}

// allocateNextMarkAndPriority allocates next available fwmark and priority.
func (r *routeBackend) allocateNextMarkAndPriority() (int, int) {
	fwMark := r.nextFwMark
	priority := r.nextPriority
	r.nextFwMark++
	r.nextPriority++

	return fwMark, priority
}

// ensureNftTableExists ensures the outway_mark chain exists in fw4 table.
func (r *routeBackend) ensureNftTableExists(ctx context.Context) {
	// Use fw4 table for OpenWrt compatibility
	tableName := Fw4TableName

	// Check if chain already exists
	cmd := exec.CommandContext(ctx, "nft", "list", "chain", "inet", tableName, "outway_mark_chain")
	if _, err := cmd.CombinedOutput(); err != nil {
		// Chain doesn't exist, create it
		cmd = exec.CommandContext(ctx, "nft", "create", "chain", "inet", tableName, "outway_mark_chain",
			"{", "type", "filter", "hook", "output", "priority", "0", ";", "}")
		if out, err := cmd.CombinedOutput(); err != nil {
			zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("table", tableName).Msg("nft create chain failed")
		} else {
			zerolog.Ctx(ctx).Debug().Str("table", tableName).Msg("nft chain created successfully")
		}
	} else {
		// Chain already exists
		zerolog.Ctx(ctx).Debug().Str("table", tableName).Msg("nft chain already exists")
	}
}

// findAvailableMarkerIP finds a free IP address in the marker pool.
func (r *routeBackend) findAvailableMarkerIP(ctx context.Context) (string, error) {
	// Parse IP range
	startIP := net.ParseIP(MarkerIPPoolStart)
	endIP := net.ParseIP(MarkerIPPoolEnd)

	if startIP == nil || endIP == nil {
		return "", fmt.Errorf("%w", ErrInvalidMarkerIPPool)
	}

	// Convert to integers for comparison
	start := ipToInt(startIP.To4())
	end := ipToInt(endIP.To4())

	// Try to find a free IP in the pool
	for ip := start; ip <= end; ip++ {
		ipStr := intToIP(ip).String()

		// Check if this IP is already used by any outway table
		if !r.isMarkerIPInUse(ctx, ipStr) {
			return ipStr, nil
		}
	}

	return "", fmt.Errorf("%w %s-%s", ErrNoFreeMarkerIP, MarkerIPPoolStart, MarkerIPPoolEnd)
}

// isMarkerIPInUse checks if a marker IP is already in use by any outway table.
func (r *routeBackend) isMarkerIPInUse(ctx context.Context, ip string) bool {
	cmd := exec.CommandContext(ctx, "ip", "route", "show", "table", "all")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return false // Assume not in use if we can't check
	}

	// Check if IP is used in any routing table
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, ip) && strings.Contains(line, "proto outway") {
			return true
		}
	}

	return false
}

// containsMarkerIPFromPool checks if a line contains any IP from the marker pool.
func (r *routeBackend) containsMarkerIPFromPool(line string) bool {
	// Parse IP range
	startIP := net.ParseIP(MarkerIPPoolStart)
	endIP := net.ParseIP(MarkerIPPoolEnd)

	if startIP == nil || endIP == nil {
		return false
	}

	// Convert to integers for comparison
	start := ipToInt(startIP.To4())
	end := ipToInt(endIP.To4())

	// Check if any IP from the pool is in the line
	for ip := start; ip <= end; ip++ {
		ipStr := intToIP(ip).String()
		if strings.Contains(line, ipStr) {
			return true
		}
	}

	return false
}

// ipToInt converts IPv4 address to integer.
func ipToInt(ip net.IP) uint32 {
	return uint32(ip[0])<<ipByteShift1 + uint32(ip[1])<<ipByteShift2 + uint32(ip[2])<<ipByteShift3 + uint32(ip[3])
}

// intToIP converts integer to IPv4 address.
func intToIP(i uint32) net.IP {
	return net.IPv4(byte(i>>ipByteShift1), byte(i>>ipByteShift2), byte(i>>ipByteShift3), byte(i))
}

// findAvailableTableID finds next available table ID with optimization.
func (r *routeBackend) findAvailableTableID(ctx context.Context) (int, error) {
	startTableID := r.nextTableID
	maxChecks := 100 // Increased from 10 to 100

	// First try sequential search
	for i := 0; i < maxChecks && startTableID+i <= RoutingTableMax; i++ {
		tableID := startTableID + i

		if r.isTableIDAvailable(ctx, tableID) {
			r.nextTableID = tableID + 1
			zerolog.Ctx(ctx).Debug().Int("tableID", tableID).Msg("found available table ID")

			return tableID, nil
		}
	}

	// Try random sampling if sequential search fails
	for i := range 20 { // Increased from 5 to 20
		tableID := RoutingTableBase + (i * tableSamplingStep)
		if tableID > RoutingTableMax {
			break
		}

		if r.isTableIDAvailable(ctx, tableID) {
			r.nextTableID = tableID + 1
			zerolog.Ctx(ctx).Debug().Int("tableID", tableID).Msg("found available table ID via sampling")

			return tableID, nil
		}
	}

	// Last resort: try all IDs in range
	zerolog.Ctx(ctx).Warn().Msg("sequential and sampling search failed, trying all IDs in range")

	for tableID := RoutingTableBase; tableID <= RoutingTableMax; tableID++ {
		if r.isTableIDAvailable(ctx, tableID) {
			r.nextTableID = tableID + 1
			zerolog.Ctx(ctx).Debug().Int("tableID", tableID).Msg("found available table ID via exhaustive search")

			return tableID, nil
		}
	}

	return 0, fmt.Errorf("%w in range %d-%d", ErrNoAvailableTableIDs, RoutingTableBase, RoutingTableMax)
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
// CRITICAL: This function must be safe and not break existing system tables.
func (r *routeBackend) cleanupOldOutwayTables(ctx context.Context) error {
	// CRITICAL: Set timeout for system commands to prevent hanging
	timeoutCtx, cancel := context.WithTimeout(ctx, systemCommandTimeout*time.Second)
	defer cancel()

	// CRITICAL: Find all routing tables safely with timeout
	cmd := exec.CommandContext(timeoutCtx, "ip", "route", "show", "table", "all")

	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("timeout listing routing tables: %w", err)
		}

		return fmt.Errorf("failed to list routing tables: %w", err)
	}

	// CRITICAL: Parse table IDs and check for outway marker safely
	lines := strings.Split(string(out), "\n")
	usedTables := make(map[int]bool)

	for _, line := range lines {
		r.extractTableIDFromLine(line, usedTables)
	}

	// CRITICAL: Check each table for outway marker and clean up safely
	cleanupErrors := make([]error, 0)

	for tableID := range usedTables {
		if r.isOutwayTable(ctx, tableID) {
			zerolog.Ctx(ctx).Info().Int("table", tableID).Msg("found old outway table, cleaning up")
			tableIDStr := strconv.Itoa(tableID)

			// CRITICAL: Clean up routes and rules safely
			r.flushTableRoutes(ctx, tableID)
			r.removeRoutingRulesForTable(ctx, tableIDStr)

			// CRITICAL: Remove nft sets for this table
			r.removeNftSetsForTable(ctx, tableID)
		}
	}

	// CRITICAL: Also clean up any tables in our range that might be empty but still exist
	zerolog.Ctx(ctx).Info().Msg("cleaning up empty tables in outway range")

	for tableID := RoutingTableBase; tableID <= RoutingTableMax; tableID++ {
		if r.isTableIDAvailable(ctx, tableID) {
			// Table is empty, but let's make sure it's really clean
			zerolog.Ctx(ctx).Debug().Int("tableID", tableID).Msg("table is empty, ensuring clean state")
			r.flushTableRoutes(ctx, tableID)
		}
	}

	// CRITICAL: Return errors if any cleanup failed
	if len(cleanupErrors) > 0 {
		return fmt.Errorf("%w: %v", ErrCleanupErrors, cleanupErrors)
	}

	return nil
}

// extractTableIDFromLine extracts table ID from a routing line and adds it to usedTables map.
func (r *routeBackend) extractTableIDFromLine(line string, usedTables map[int]bool) {
	if !strings.Contains(line, "table") {
		return
	}

	// Extract table ID from lines like "default via 192.168.1.1 dev eth0 table 100"
	parts := strings.Fields(line)
	for i, part := range parts {
		if part == tableKeyword && i+1 < len(parts) {
			if tableID, err := strconv.Atoi(parts[i+1]); err == nil {
				// CRITICAL: Only consider tables in our range
				if tableID >= RoutingTableBase && tableID <= RoutingTableMax {
					usedTables[tableID] = true
				}
			}
		}
	}
}

// removeNftSetsForTable removes nft sets for a specific table.
// CRITICAL: This function must be safe and handle errors properly.
func (r *routeBackend) removeNftSetsForTable(ctx context.Context, tableID int) {
	// CRITICAL: Try to find tunnel name for this table ID
	tunnelName := ""

	for name, info := range r.tunnels {
		if info.TableID == tableID {
			tunnelName = name

			break
		}
	}

	if tunnelName == "" {
		// CRITICAL: If we can't find the tunnel, try generic cleanup
		tunnelName = fmt.Sprintf("table_%d", tableID)
	}

	// Use fw4 table for OpenWrt compatibility
	tableName := Fw4TableName

	// CRITICAL: Remove both IPv4 and IPv6 sets
	cmd := exec.CommandContext(ctx, "nft", "delete", "set", "inet", tableName, "outway_"+tunnelName+"_4")
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("tunnel", tunnelName).Msg("nft delete set IPv4 (may not exist)")
	}

	cmd = exec.CommandContext(ctx, "nft", "delete", "set", "inet", tableName, "outway_"+tunnelName+"_6")
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("tunnel", tunnelName).Msg("nft delete set IPv6 (may not exist)")
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

func (r *routeBackend) validateMarkIPInputs(iface, ip string) error {
	if !IsSafeIfaceName(iface) {
		return fmt.Errorf("%w: %q", ErrInvalidIface, iface)
	}

	if _, ok := NormalizeIP(ip); !ok {
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

	// Get tunnel info to remove route from correct table
	tunnel, exists := r.tunnels[iface]
	if !exists {
		zerolog.Ctx(ctx).Warn().Str("iface", iface).Msg("tunnel not found for route removal")
		delete(r.entries, ip)

		return
	}

	// Remove route for this IP from the tunnel's table
	tableIDStr := strconv.Itoa(tunnel.TableID)

	cmd := exec.CommandContext(ctx, "ip", "route", "del", ip+"/32", "dev", iface, "table", tableIDStr) //nolint:gosec // ip is validated
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("ip", ip).Str("iface", iface).Str("table", tableIDStr).Msg("ip route del (may not exist)")
	} else {
		zerolog.Ctx(ctx).Debug().Str("ip", ip).Str("iface", iface).Str("table", tableIDStr).Msg("route removed successfully")
	}

	delete(r.entries, ip)
}

// IsSafeIfaceName verifies interface names to a conservative charset to avoid injection via args.
var IfaceNameRe = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,32}$`)

func IsSafeIfaceName(iface string) bool {
	return IfaceNameRe.MatchString(iface)
}

// NormalizeIP parses and returns canonical string representation without brackets.
func NormalizeIP(raw string) (string, bool) {
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

func IsIPv6(ip string) bool {
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

// findOrCreateTableForTunnel finds an existing outway table or creates a new one.
// CRITICAL: This function must be atomic and thread-safe.
func (r *routeBackend) findOrCreateTableForTunnel(ctx context.Context, tunnel string) (int, error) {
	zerolog.Ctx(ctx).Info().Str("tunnel", tunnel).Msg("searching for table for tunnel")

	// CRITICAL: First, try to find existing outway table for this specific tunnel
	existingTable, err := r.findExistingOutwayTableForTunnel(ctx, tunnel)
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Str("tunnel", tunnel).Msg("failed to search for existing table")

		return 0, fmt.Errorf("failed to search for existing table: %w", err)
	}

	if existingTable != 0 {
		zerolog.Ctx(ctx).Info().Str("tunnel", tunnel).Int("tableID", existingTable).Msg("found existing table for tunnel")

		return existingTable, nil
	}

	// CRITICAL: Find next available table ID atomically
	zerolog.Ctx(ctx).Info().Str("tunnel", tunnel).Msg("no existing table found, searching for available table ID")

	tableID, err := r.findAvailableTableID(ctx)
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Str("tunnel", tunnel).Msg("failed to find available table ID")

		return 0, err
	}

	zerolog.Ctx(ctx).Info().Str("tunnel", tunnel).Int("tableID", tableID).Msg("found available table ID for tunnel")

	return tableID, nil
}

// findExistingOutwayTableForTunnel looks for existing outway table for a specific tunnel.
// CRITICAL: This function must be tunnel-specific to avoid conflicts.
func (r *routeBackend) findExistingOutwayTableForTunnel(ctx context.Context, tunnel string) (int, error) {
	cmd := exec.CommandContext(ctx, "ip", "route", "show", "table", "all")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to list routing tables: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if tableID := r.extractOutwayTableIDFromLine(ctx, line, tunnel); tableID != 0 {
			return tableID, nil
		}
	}

	return 0, nil
}

// extractOutwayTableIDFromLine extracts outway table ID from a routing line for a specific tunnel.
func (r *routeBackend) extractOutwayTableIDFromLine(ctx context.Context, line, tunnel string) int {
	// Look for outway marker with specific tunnel interface
	if !strings.Contains(line, "table") || !strings.Contains(line, tunnel) {
		return 0
	}

	// Check if this line contains any IP from our marker pool
	if !r.containsMarkerIPFromPool(line) {
		return 0
	}

	// Extract table ID
	parts := strings.Fields(line)
	for i, part := range parts {
		if part == "table" && i+1 < len(parts) {
			if tableID, err := strconv.Atoi(parts[i+1]); err == nil {
				// CRITICAL: Double-check this is an outway table for this tunnel
				if r.isOutwayTableForTunnel(ctx, tableID, tunnel) {
					return tableID
				}
			}
		}
	}

	return 0
}

// isTableIDAvailable checks if a table ID is available.
func (r *routeBackend) isTableIDAvailable(ctx context.Context, tableID int) bool {
	cmd := exec.CommandContext(ctx, "ip", "route", "show", "table", strconv.Itoa(tableID)) //nolint:gosec

	out, err := cmd.CombinedOutput()
	if err != nil {
		// Table doesn't exist, so it's available
		zerolog.Ctx(ctx).Debug().Int("tableID", tableID).Msg("table doesn't exist, available")

		return true
	}

	// Table is available if it's empty (no routes)
	output := strings.TrimSpace(string(out))
	isEmpty := len(output) == 0

	zerolog.Ctx(ctx).Debug().Int("tableID", tableID).Bool("isEmpty", isEmpty).Str("output", output).Msg("table availability check")

	return isEmpty
}

// isOutwayTableForTunnel checks if a table is an outway table for a specific tunnel.
func (r *routeBackend) isOutwayTableForTunnel(ctx context.Context, tableID int, tunnel string) bool {
	cmd := exec.CommandContext(ctx, "ip", "route", "show", "table", strconv.Itoa(tableID)) //nolint:gosec // tableID is validated integer

	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	// Check for outway marker route with specific tunnel interface
	output := string(out)

	return r.containsMarkerIPFromPool(output) &&
		strings.Contains(output, OutwayMarkerProto) &&
		strings.Contains(output, strconv.Itoa(OutwayMarkerMetric)) &&
		strings.Contains(output, tunnel)
}

// setupTunnelTable sets up routing table and rules for a tunnel.
func (r *routeBackend) setupTunnelTable(ctx context.Context, info TunnelInfo, markerIP string) {
	tableID := strconv.Itoa(info.TableID)

	// Create default route through the interface
	cmd := exec.CommandContext(ctx, "ip", "route", "add", "default", "dev", info.Name, "table", tableID) //nolint:gosec // validated
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("iface", info.Name).Str("table", tableID).Msg("ip route add default (may already exist)")
	}

	// Add outway marker route with all parameters
	args := []string{
		"route", "add", markerIP + "/32", "dev", "lo", "table", tableID,
		"metric", strconv.Itoa(OutwayMarkerMetric),
	}

	cmd = exec.CommandContext(ctx, "ip", args...) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("table", tableID).Msg("ip route add marker (may already exist)")
	}

	// Create ip rule for fwmark -> table mapping
	ruleArgs := []string{"rule", "add", "fwmark", strconv.Itoa(info.FwMark), "table", tableID, "priority", strconv.Itoa(info.Priority)}

	cmd = exec.CommandContext(ctx, "ip", ruleArgs...) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Int("fwmark", info.FwMark).Str("table", tableID).Msg("ip rule add (may already exist)")
	}

	// Create nft sets for this tunnel
	r.createNftSets(ctx, info.Name)

	// Create nftables rules for fwmark
	r.createNftRules(ctx, info.Name, info.FwMark)

	// Force create nftables table and chain if they don't exist
	r.ensureNftTableExists(ctx)

	// Force create rules again to ensure they exist
	r.forceCreateNftRules(ctx, info.Name, info.FwMark)

	// Verify rules exist and create if missing
	r.verifyAndCreateNftRules(ctx, info.Name, info.FwMark)

	// Log successful tunnel setup
	zerolog.Ctx(ctx).Info().
		Str("tunnel", info.Name).
		Int("table", info.TableID).
		Int("fwmark", info.FwMark).
		Int("priority", info.Priority).
		Str("marker_ip", markerIP).
		Msg("tunnel initialized successfully")
}

// createNftSets creates nftables sets for a tunnel.
func (r *routeBackend) createNftSets(ctx context.Context, tunnel string) {
	// Use fw4 table for OpenWrt compatibility
	tableName := Fw4TableName

	// Create IPv4 set with timeout flags
	setArgs4 := []string{
		"add", "set", "inet", tableName, "outway_" + tunnel + "_4", "{", "type", "ipv4_addr", ";", "flags", "timeout", ";", "}",
	}

	cmd := exec.CommandContext(ctx, "nft", setArgs4...) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("tunnel", tunnel).Msg("nft add set IPv4 (may already exist)")
	}

	// Create IPv6 set with timeout flags
	setArgs6 := []string{
		"add", "set", "inet", tableName, "outway_" + tunnel + "_6", "{", "type", "ipv6_addr", ";", "flags", "timeout", ";", "}",
	}

	cmd = exec.CommandContext(ctx, "nft", setArgs6...) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("tunnel", tunnel).Msg("nft add set IPv6 (may already exist)")
	}
}

// createNftRules creates nftables rules for setting fwmark.
func (r *routeBackend) createNftRules(ctx context.Context, tunnel string, fwMark int) {
	// Ensure table and chain exist first
	r.ensureNftTableExists(ctx)

	// Use fw4 table for OpenWrt compatibility
	tableName := Fw4TableName

	// Always create IPv4 rule (ignore if already exists)
	cmd := exec.CommandContext(ctx, "nft", "add", "rule", "inet", tableName, "outway_mark_chain", //nolint:gosec
		"ip", "daddr", "@outway_"+tunnel+"_4", "meta", "mark", "set", strconv.Itoa(fwMark))
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("tunnel", tunnel).Msg("nft add rule IPv4 (may already exist)")
	} else {
		zerolog.Ctx(ctx).Info().Str("tunnel", tunnel).Int("fwmark", fwMark).Msg("nft rule IPv4 created successfully")
	}

	// Always create IPv6 rule (ignore if already exists)
	cmd = exec.CommandContext(ctx, "nft", "add", "rule", "inet", tableName, "outway_mark_chain", //nolint:gosec
		"ip6", "daddr", "@outway_"+tunnel+"_6", "meta", "mark", "set", strconv.Itoa(fwMark))
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("tunnel", tunnel).Msg("nft add rule IPv6 (may already exist)")
	} else {
		zerolog.Ctx(ctx).Info().Str("tunnel", tunnel).Int("fwmark", fwMark).Msg("nft rule IPv6 created successfully")
	}
}

// forceCreateNftRules forces creation of nftables rules with error handling.
func (r *routeBackend) forceCreateNftRules(ctx context.Context, tunnel string, fwMark int) {
	// First ensure table and chain exist
	r.ensureNftTableExists(ctx)

	// Use fw4 table for OpenWrt compatibility
	tableName := Fw4TableName

	// Force create IPv4 rule
	cmd := exec.CommandContext(ctx, "nft", "add", "rule", "inet", tableName, "outway_mark_chain", //nolint:gosec
		"ip", "daddr", "@outway_"+tunnel+"_4", "meta", "mark", "set", strconv.Itoa(fwMark))
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Warn().Bytes("out", out).Str("tunnel", tunnel).Int("fwmark", fwMark).Msg("failed to create IPv4 rule")
	} else {
		zerolog.Ctx(ctx).Info().Str("tunnel", tunnel).Int("fwmark", fwMark).Msg("IPv4 rule created successfully")
	}

	// Force create IPv6 rule
	cmd = exec.CommandContext(ctx, "nft", "add", "rule", "inet", tableName, "outway_mark_chain", //nolint:gosec
		"ip6", "daddr", "@outway_"+tunnel+"_6", "meta", "mark", "set", strconv.Itoa(fwMark))
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Warn().Bytes("out", out).Str("tunnel", tunnel).Int("fwmark", fwMark).Msg("failed to create IPv6 rule")
	} else {
		zerolog.Ctx(ctx).Info().Str("tunnel", tunnel).Int("fwmark", fwMark).Msg("IPv6 rule created successfully")
	}
}

// verifyAndCreateNftRules verifies that nftables rules exist and creates them if missing.
func (r *routeBackend) verifyAndCreateNftRules(ctx context.Context, tunnel string, fwMark int) {
	// Use fw4 table for OpenWrt compatibility
	tableName := Fw4TableName

	// Check if IPv4 rule exists
	cmd := exec.CommandContext(ctx, "nft", "list", "chain", "inet", tableName, "outway_mark_chain")
	if out, err := cmd.CombinedOutput(); err == nil {
		rulePattern := fmt.Sprintf("ip daddr @outway_%s_4 meta mark set %d", tunnel, fwMark)
		if !strings.Contains(string(out), rulePattern) {
			zerolog.Ctx(ctx).Warn().Str("tunnel", tunnel).Int("fwmark", fwMark).Msg("IPv4 rule missing, creating...")
			// Create IPv4 rule
			cmd = exec.CommandContext(ctx, "nft", "add", "rule", "inet", tableName, "outway_mark_chain", //nolint:gosec
				"ip", "daddr", "@outway_"+tunnel+"_4", "meta", "mark", "set", strconv.Itoa(fwMark))
			if out, err := cmd.CombinedOutput(); err != nil {
				zerolog.Ctx(ctx).Error().Bytes("out", out).Str("tunnel", tunnel).Int("fwmark", fwMark).Msg("failed to create IPv4 rule")
			} else {
				zerolog.Ctx(ctx).Info().Str("tunnel", tunnel).Int("fwmark", fwMark).Msg("IPv4 rule created successfully")
			}
		}
	}

	// Check if IPv6 rule exists
	cmd = exec.CommandContext(ctx, "nft", "list", "chain", "inet", tableName, "outway_mark_chain")
	if out, err := cmd.CombinedOutput(); err == nil {
		rulePattern := fmt.Sprintf("ip6 daddr @outway_%s_6 meta mark set %d", tunnel, fwMark)
		if !strings.Contains(string(out), rulePattern) {
			zerolog.Ctx(ctx).Warn().Str("tunnel", tunnel).Int("fwmark", fwMark).Msg("IPv6 rule missing, creating...")
			// Create IPv6 rule
			cmd = exec.CommandContext(ctx, "nft", "add", "rule", "inet", tableName, "outway_mark_chain", //nolint:gosec
				"ip6", "daddr", "@outway_"+tunnel+"_6", "meta", "mark", "set", strconv.Itoa(fwMark))
			if out, err := cmd.CombinedOutput(); err != nil {
				zerolog.Ctx(ctx).Error().Bytes("out", out).Str("tunnel", tunnel).Int("fwmark", fwMark).Msg("failed to create IPv6 rule")
			} else {
				zerolog.Ctx(ctx).Info().Str("tunnel", tunnel).Int("fwmark", fwMark).Msg("IPv6 rule created successfully")
			}
		}
	}
}

// addIPToNftSet adds an IP to the appropriate nft set.
func (r *routeBackend) addIPToNftSet(ctx context.Context, ip, tunnel string, ttlSeconds int) {
	setName := "outway_" + tunnel + "_4"
	if IsIPv6(ip) {
		setName = "outway_" + tunnel + "_6"
	}

	// Use fw4 table for OpenWrt compatibility
	tableName := Fw4TableName

	// Add IP element to nft set with timeout
	elemArgs := []string{"add", "element", "inet", tableName, setName, "{", ip, "timeout", strconv.Itoa(ttlSeconds) + "s", "}"}

	cmd := exec.CommandContext(ctx, "nft", elemArgs...) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("ip", ip).Str("set", setName).Msg("nft add element (may already exist)")
	}
}

// createRouteForIPInTable creates a route for IP in specific table.
func (r *routeBackend) createRouteForIPInTable(ctx context.Context, ip, iface string, tableID int) error {
	tableIDStr := strconv.Itoa(tableID)
	cmd := exec.CommandContext(ctx, "ip", "route", "add", ip+"/32", "dev", iface, "table", tableIDStr) //nolint:gosec

	if out, err := cmd.CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "File exists") {
			zerolog.Ctx(ctx).Debug().
				Bytes("out", out).
				Str("ip", ip).
				Str("iface", iface).
				Str("table", tableIDStr).
				Msg("ip route add (may already exist)")

			return fmt.Errorf("failed to add route for %s: %w", ip, err)
		}

		zerolog.Ctx(ctx).Debug().Str("ip", ip).Str("iface", iface).Str("table", tableIDStr).Msg("route already exists")
	} else {
		zerolog.Ctx(ctx).Debug().Str("ip", ip).Str("iface", iface).Str("table", tableIDStr).Msg("route created successfully")
	}

	return nil
}

// flushTableRoutes flushes routes from a table but keeps the table structure.
func (r *routeBackend) flushTableRoutes(ctx context.Context, tableID int) {
	cmd := exec.CommandContext(ctx, "ip", "route", "flush", "table", strconv.Itoa(tableID)) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Int("table", tableID).Msg("ip route flush table (may not exist)")
	}
}

// flushNftSets flushes nft sets for a tunnel.
func (r *routeBackend) flushNftSets(ctx context.Context, tunnel string) {
	// Use fw4 table for OpenWrt compatibility
	tableName := Fw4TableName

	// Flush IPv4 set
	cmd := exec.CommandContext(ctx, "nft", "flush", "set", "inet", tableName, "outway_"+tunnel+"_4") //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("tunnel", tunnel).Msg("nft flush set IPv4 (may not exist)")
	}

	// Flush IPv6 set
	cmd = exec.CommandContext(ctx, "nft", "flush", "set", "inet", tableName, "outway_"+tunnel+"_6") //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("tunnel", tunnel).Msg("nft flush set IPv6 (may not exist)")
	}
}

// removeNftSets removes nft sets for a tunnel.
func (r *routeBackend) removeNftSets(ctx context.Context, tunnel string) {
	// Use fw4 table for OpenWrt compatibility
	tableName := Fw4TableName

	// Remove IPv4 set
	cmd := exec.CommandContext(ctx, "nft", "delete", "set", "inet", tableName, "outway_"+tunnel+"_4") //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("tunnel", tunnel).Msg("nft delete set IPv4 (may not exist)")
	}

	// Remove IPv6 set
	cmd = exec.CommandContext(ctx, "nft", "delete", "set", "inet", tableName, "outway_"+tunnel+"_6") //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("tunnel", tunnel).Msg("nft delete set IPv6 (may not exist)")
	}
}

// removeRoutingRulesForTable removes all rules pointing to a specific table.
func (r *routeBackend) removeRoutingRulesForTable(ctx context.Context, tableID string) {
	cmd := exec.CommandContext(ctx, "ip", "rule", "show")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "table "+tableID) {
			r.deleteRuleForTable(ctx, line, tableID)
		}
	}
}

// deleteRuleForTable deletes a specific routing rule for a table.
func (r *routeBackend) deleteRuleForTable(ctx context.Context, line, tableID string) {
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
