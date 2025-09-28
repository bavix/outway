package firewall

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// ErrNoAvailableMarkerIP is returned when no available marker IP is found.
var ErrNoAvailableMarkerIP = errors.New("no available marker IP found")

// DynamicRouteBackendV2 is an improved route backend with batching and dynamic allocation.
type DynamicRouteBackendV2 struct {
	mutex        sync.RWMutex
	entries      map[string]routeEntry
	tunnels      map[string]DynamicTunnelInfo
	nextTableID  int
	nextFwMark   int
	nextPriority int
}

// DynamicTunnelInfo represents enhanced tunnel information with dynamic allocation.
type DynamicTunnelInfo struct {
	Name     string
	TableID  int
	FwMark   int
	Priority int
	IPv4Set  string
	IPv6Set  string
}

// NewDynamicRouteBackendV2 creates a new dynamic route backend.
func NewDynamicRouteBackendV2() *DynamicRouteBackendV2 {
	return &DynamicRouteBackendV2{
		entries:      make(map[string]routeEntry),
		tunnels:      make(map[string]DynamicTunnelInfo),
		nextTableID:  RoutingTableBase,
		nextFwMark:   RoutingTableBase,
		nextPriority: RoutingTableBase,
	}
}

func (r *DynamicRouteBackendV2) Name() string { return "dynamic_route" }

// InitializeTunnels initializes tunnel interfaces with dynamic fwmark/table allocation.
//
//nolint:funlen
func (r *DynamicRouteBackendV2) InitializeTunnels(ctx context.Context, tunnels []string) ([]TunnelInfo, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Validate tunnel list
	for _, tunnel := range tunnels {
		if !IsSafeIfaceName(tunnel) {
			return nil, fmt.Errorf("%w: %q", ErrInvalidIface, tunnel)
		}
	}

	log := zerolog.Ctx(ctx)
	tunnelInfos := make([]TunnelInfo, 0, len(tunnels))

	// Clean up old tables first
	r.cleanupOldTables(ctx)

	// Create outway table if it doesn't exist
	if err := r.ensureOutwayTable(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure outway table: %w", err)
	}

	for _, tunnel := range tunnels {
		// Check if tunnel already exists
		if info, exists := r.tunnels[tunnel]; exists {
			tunnelInfos = append(tunnelInfos, TunnelInfo{
				Name:     info.Name,
				TableID:  info.TableID,
				FwMark:   info.FwMark,
				Priority: info.Priority,
			})

			continue
		}

		// Allocate dynamic IDs
		tableID := r.allocateNextTableID()
		fwMark := r.allocateNextFwMark()
		priority := r.allocateNextPriority()

		// Create tunnel info
		info := DynamicTunnelInfo{
			Name:     tunnel,
			TableID:  tableID,
			FwMark:   fwMark,
			Priority: priority,
			IPv4Set:  fmt.Sprintf("tun_%s_v4", tunnel),
			IPv6Set:  fmt.Sprintf("tun_%s_v6", tunnel),
		}

		// Setup tunnel infrastructure
		r.setupTunnelInfrastructure(ctx, &info)

		// Setup routing table and rules
		markerIP, err := r.findAvailableMarkerIP(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to find available marker IP: %w", err)
		}

		r.setupTunnelTable(ctx, info, markerIP)

		// Store tunnel info
		r.tunnels[tunnel] = info
		tunnelInfos = append(tunnelInfos, TunnelInfo{
			Name:     info.Name,
			TableID:  info.TableID,
			FwMark:   info.FwMark,
			Priority: info.Priority,
		})

		log.Info().
			Str("tunnel", tunnel).
			Int("table", tableID).
			Int("fwmark", fwMark).
			Int("priority", priority).
			Msg("tunnel initialized with dynamic allocation")
	}

	return tunnelInfos, nil
}

// ensureOutwayTable ensures the outway table exists.
//
//nolint:funcorder
func (r *DynamicRouteBackendV2) ensureOutwayTable(ctx context.Context) error {
	// Check if table already exists
	cmd := exec.CommandContext(ctx, "nft", "list", "tables")

	out, err := cmd.CombinedOutput()
	if err != nil {
		zerolog.Ctx(ctx).Warn().Err(err).Msg("failed to list nftables tables")
	}

	if !strings.Contains(string(out), "table inet outway") {
		// Create outway table
		cmd = exec.CommandContext(ctx, "nft", "create", "table", "inet", "outway")
		if out, err := cmd.CombinedOutput(); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Bytes("out", out).Msg("failed to create outway table")

			return fmt.Errorf("failed to create outway table: %w", err)
		}

		zerolog.Ctx(ctx).Info().Msg("outway table created successfully")
	} else {
		zerolog.Ctx(ctx).Debug().Msg("outway table already exists")
	}

	// Create output chain if it doesn't exist
	cmd = exec.CommandContext(ctx, "nft", "list", "chain", "inet", "outway", "output")
	if _, err := cmd.CombinedOutput(); err != nil {
		// Chain doesn't exist, create it
		cmd = exec.CommandContext(ctx, "nft", "create", "chain", "inet", "outway", "output",
			"{", "type", "filter", "hook", "output", "priority", "0", ";", "}")
		if out, err := cmd.CombinedOutput(); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Bytes("out", out).Msg("failed to create output chain")

			return fmt.Errorf("failed to create output chain: %w", err)
		}

		zerolog.Ctx(ctx).Info().Msg("output chain created successfully")
	} else {
		zerolog.Ctx(ctx).Debug().Msg("output chain already exists")
	}

	return nil
}

// setupTunnelInfrastructure sets up nftables infrastructure for a tunnel.
//
//nolint:funcorder
func (r *DynamicRouteBackendV2) setupTunnelInfrastructure(ctx context.Context, info *DynamicTunnelInfo) {
	// Create IPv4 set with timeout
	cmd := exec.CommandContext(ctx, "nft", "add", "set", "inet", "outway", info.IPv4Set, //nolint:gosec
		"{", "type", "ipv4_addr", ";", "flags", "timeout", ";", "}")
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Warn().Bytes("out", out).Str("set", info.IPv4Set).Msg("failed to create IPv4 set (may already exist)")
	} else {
		zerolog.Ctx(ctx).Debug().Str("set", info.IPv4Set).Msg("IPv4 set created successfully")
	}

	// Create IPv6 set with timeout
	cmd = exec.CommandContext(ctx, "nft", "add", "set", "inet", "outway", info.IPv6Set, //nolint:gosec
		"{", "type", "ipv6_addr", ";", "flags", "timeout", ";", "}")
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Warn().Bytes("out", out).Str("set", info.IPv6Set).Msg("failed to create IPv6 set (may already exist)")
	} else {
		zerolog.Ctx(ctx).Debug().Str("set", info.IPv6Set).Msg("IPv6 set created successfully")
	}

	// Create marking rules
	r.createMarkingRules(ctx, info)

	zerolog.Ctx(ctx).Info().
		Str("tunnel", info.Name).
		Int("fwmark", info.FwMark).
		Msg("tunnel infrastructure setup completed")
}

// createMarkingRules creates nftables rules for marking packets.
//
//nolint:funcorder
func (r *DynamicRouteBackendV2) createMarkingRules(ctx context.Context, info *DynamicTunnelInfo) {
	// Create IPv4 marking rule
	cmd := exec.CommandContext(ctx, "nft", "add", "rule", "inet", "outway", "output", //nolint:gosec
		"ip", "daddr", "@"+info.IPv4Set, "meta", "mark", "set", strconv.Itoa(info.FwMark))
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Warn().Bytes("out", out).Str("tunnel", info.Name).Msg("failed to create IPv4 marking rule (may already exist)")
	} else {
		zerolog.Ctx(ctx).Debug().Str("tunnel", info.Name).Msg("IPv4 marking rule created")
	}

	// Create IPv6 marking rule
	cmd = exec.CommandContext(ctx, "nft", "add", "rule", "inet", "outway", "output", //nolint:gosec
		"ip6", "daddr", "@"+info.IPv6Set, "meta", "mark", "set", strconv.Itoa(info.FwMark))
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Warn().Bytes("out", out).Str("tunnel", info.Name).Msg("failed to create IPv6 marking rule (may already exist)")
	} else {
		zerolog.Ctx(ctx).Debug().Str("tunnel", info.Name).Msg("IPv6 marking rule created")
	}

	zerolog.Ctx(ctx).Info().
		Str("tunnel", info.Name).
		Int("fwmark", info.FwMark).
		Msg("marking rules created successfully")
}

// setupTunnelTable sets up routing table and rules for a tunnel.
//
//nolint:funcorder
func (r *DynamicRouteBackendV2) setupTunnelTable(ctx context.Context, info DynamicTunnelInfo, markerIP string) {
	tableID := strconv.Itoa(info.TableID)

	// Create default route through the interface
	cmd := exec.CommandContext(ctx, "ip", "route", "add", "default", "dev", info.Name, "table", tableID) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("iface", info.Name).Str("table", tableID).Msg("ip route add default (may already exist)")
	}

	// Add outway marker route
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

	zerolog.Ctx(ctx).Info().
		Str("tunnel", info.Name).
		Int("table", info.TableID).
		Int("fwmark", info.FwMark).
		Int("priority", info.Priority).
		Str("marker_ip", markerIP).
		Msg("tunnel table setup completed")
}

// MarkIP marks an IP address for routing through a specific tunnel.
func (r *DynamicRouteBackendV2) MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	info, exists := r.tunnels[iface]
	if !exists {
		return fmt.Errorf("%w: %s", ErrTunnelNotFound, iface)
	}

	// Determine if IPv4 or IPv6
	isIPv6 := IsIPv6(ip)

	var setName string
	if isIPv6 {
		setName = info.IPv6Set
	} else {
		setName = info.IPv4Set
	}

	// Add IP to set with TTL
	cmd := exec.CommandContext(ctx, "nft", "add", "element", "inet", "outway", setName, //nolint:gosec
		"{", ip, "timeout", strconv.Itoa(ttlSeconds)+"s", "}")
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Bytes("out", out).Str("ip", ip).Str("set", setName).Msg("failed to add IP to set")

		return fmt.Errorf("failed to add IP %s to set %s: %w", ip, setName, err)
	}

	// Store entry for tracking
	r.entries[ip] = routeEntry{
		expireAt: time.Now().Add(time.Duration(ttlSeconds) * time.Second),
		iface:    iface,
	}

	zerolog.Ctx(ctx).Info().
		Str("ip", ip).
		Str("tunnel", iface).
		Int("ttl", ttlSeconds).
		Bool("ipv6", isIPv6).
		Msg("IP marked for tunnel routing")

	return nil
}

// MarkIPBatch marks multiple IP addresses in a single batch operation.
func (r *DynamicRouteBackendV2) MarkIPBatch(ctx context.Context, iface string, ips []string, ttlSeconds int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	info, exists := r.tunnels[iface]
	if !exists {
		return fmt.Errorf("%w: %s", ErrTunnelNotFound, iface)
	}

	// Separate IPv4 and IPv6 IPs
	ipv4IPs := make([]string, 0)
	ipv6IPs := make([]string, 0)

	for _, ip := range ips {
		if IsIPv6(ip) {
			ipv6IPs = append(ipv6IPs, ip)
		} else {
			ipv4IPs = append(ipv4IPs, ip)
		}
	}

	// Add IPv4 IPs to set in batch
	if len(ipv4IPs) > 0 {
		cmd := exec.CommandContext(ctx, "nft", "add", "element", "inet", "outway", info.IPv4Set, //nolint:gosec
			"{", strings.Join(ipv4IPs, " timeout "+strconv.Itoa(ttlSeconds)+"s , "), "}")
		if out, err := cmd.CombinedOutput(); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Bytes("out", out).Str("tunnel", iface).Msg("failed to add IPv4 IPs to set")
		} else {
			zerolog.Ctx(ctx).Debug().Str("tunnel", iface).Int("count", len(ipv4IPs)).Msg("IPv4 IPs added to set")
		}
	}

	// Add IPv6 IPs to set in batch
	if len(ipv6IPs) > 0 {
		cmd := exec.CommandContext(ctx, "nft", "add", "element", "inet", "outway", info.IPv6Set, //nolint:gosec
			"{", strings.Join(ipv6IPs, " timeout "+strconv.Itoa(ttlSeconds)+"s , "), "}")
		if out, err := cmd.CombinedOutput(); err != nil {
			zerolog.Ctx(ctx).Error().Err(err).Bytes("out", out).Str("tunnel", iface).Msg("failed to add IPv6 IPs to set")
		} else {
			zerolog.Ctx(ctx).Debug().Str("tunnel", iface).Int("count", len(ipv6IPs)).Msg("IPv6 IPs added to set")
		}
	}

	// Store entries for tracking
	now := time.Now()

	ttl := time.Duration(ttlSeconds) * time.Second
	for _, ip := range ips {
		r.entries[ip] = routeEntry{
			expireAt: now.Add(ttl),
			iface:    iface,
		}
	}

	zerolog.Ctx(ctx).Info().
		Str("tunnel", iface).
		Int("total_ips", len(ips)).
		Int("ipv4_ips", len(ipv4IPs)).
		Int("ipv6_ips", len(ipv6IPs)).
		Int("ttl", ttlSeconds).
		Msg("batch IP marking completed")

	return nil
}

// GetTunnelInfo returns information about a specific tunnel.
func (r *DynamicRouteBackendV2) GetTunnelInfo(ctx context.Context, iface string) (*TunnelInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	info, exists := r.tunnels[iface]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTunnelNotFound, iface)
	}

	return &TunnelInfo{
		Name:     info.Name,
		TableID:  info.TableID,
		FwMark:   info.FwMark,
		Priority: info.Priority,
	}, nil
}

// FlushRuntime flushes runtime data from tables but keeps the table structure.
func (r *DynamicRouteBackendV2) FlushRuntime(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log := zerolog.Ctx(ctx)
	log.Info().Msg("flushing runtime data from outway tables")

	for _, tunnel := range r.tunnels {
		// Flush routes from table but keep the table structure
		r.flushTableRoutes(ctx, tunnel.TableID)

		// Note: nftables sets with timeout will automatically expire
		// No need to manually flush them
	}

	// Clear local entries
	r.entries = make(map[string]routeEntry)

	return nil
}

// CleanupAll removes all outway tables and rules.
func (r *DynamicRouteBackendV2) CleanupAll(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log := zerolog.Ctx(ctx)
	log.Info().Msg("cleaning up all outway tables and rules")

	// Delete outway table (this will remove all sets, chains, and rules)
	cmd := exec.CommandContext(ctx, "nft", "delete", "table", "inet", "outway")
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Warn().Err(err).Bytes("out", out).Msg("failed to delete outway table")
	} else {
		log.Info().Msg("outway table deleted successfully")
	}

	// Clean up routing tables and rules
	for _, tunnel := range r.tunnels {
		r.cleanupTunnelTable(ctx, tunnel)
	}

	// Clear all data
	r.tunnels = make(map[string]DynamicTunnelInfo)
	r.entries = make(map[string]routeEntry)

	log.Info().Msg("outway cleanup completed")

	return nil
}

// cleanupTunnelTable cleans up routing table and rules for a tunnel.
func (r *DynamicRouteBackendV2) cleanupTunnelTable(ctx context.Context, info DynamicTunnelInfo) {
	tableID := strconv.Itoa(info.TableID)

	// Remove routing rules
	cmd := exec.CommandContext(ctx, "ip", "rule", "del", "fwmark", strconv.Itoa(info.FwMark), "table", tableID) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Int("fwmark", info.FwMark).Msg("ip rule del (may not exist)")
	}

	// Flush routing table
	r.flushTableRoutes(ctx, info.TableID)

	zerolog.Ctx(ctx).Info().
		Str("tunnel", info.Name).
		Int("table", info.TableID).
		Int("fwmark", info.FwMark).
		Msg("tunnel table cleaned up")
}

// Helper methods for ID allocation.
func (r *DynamicRouteBackendV2) allocateNextTableID() int {
	id := r.nextTableID
	r.nextTableID++

	return id
}

func (r *DynamicRouteBackendV2) allocateNextFwMark() int {
	mark := r.nextFwMark
	r.nextFwMark++

	return mark
}

func (r *DynamicRouteBackendV2) allocateNextPriority() int {
	priority := r.nextPriority
	r.nextPriority++

	return priority
}

// cleanupOldTables cleans up old outway tables.
func (r *DynamicRouteBackendV2) cleanupOldTables(ctx context.Context) {
	// Delete outway table if it exists
	cmd := exec.CommandContext(ctx, "nft", "delete", "table", "inet", "outway")
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Msg("outway table cleanup (may not exist)")
	}
}

// findAvailableMarkerIP finds an available marker IP for routing.
func (r *DynamicRouteBackendV2) findAvailableMarkerIP(ctx context.Context) (string, error) {
	// Use a simple approach - try common marker IPs
	markerIPs := []string{
		"169.254.1.1",
		"169.254.1.2",
		"169.254.1.3",
		"169.254.1.4",
		"169.254.1.5",
	}

	for _, ip := range markerIPs {
		// Check if IP is already in use
		cmd := exec.CommandContext(ctx, "ip", "route", "get", ip) //nolint:gosec

		out, err := cmd.CombinedOutput()
		if err != nil {
			// IP is not reachable, we can use it
			zerolog.Ctx(ctx).Debug().Str("marker_ip", ip).Msg("found available marker IP")

			return ip, nil //nolint:nilerr
		}

		// IP is reachable, check if it's already in use
		if len(out) > 0 {
			zerolog.Ctx(ctx).Debug().Str("marker_ip", ip).Msg("marker IP already in use")

			continue
		}

		// IP is available
		zerolog.Ctx(ctx).Debug().Str("marker_ip", ip).Msg("found available marker IP")

		return ip, nil
	}

	return "", ErrNoAvailableMarkerIP
}

// flushTableRoutes flushes all routes from a routing table.
func (r *DynamicRouteBackendV2) flushTableRoutes(ctx context.Context, tableID int) {
	tableIDStr := strconv.Itoa(tableID)

	cmd := exec.CommandContext(ctx, "ip", "route", "flush", "table", tableIDStr) //nolint:gosec
	if out, err := cmd.CombinedOutput(); err != nil {
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Str("table", tableIDStr).Msg("ip route flush table (may not exist)")
	}
}
