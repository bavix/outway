package firewall

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// SimpleRouteBackend uses ip route expires for automatic cleanup
type SimpleRouteBackend struct {
	mutex   sync.RWMutex
	entries map[string]time.Time // Track when routes expire to avoid duplicates
}

// NewSimpleRouteBackend creates a new simple route backend
func NewSimpleRouteBackend() *SimpleRouteBackend {
	return &SimpleRouteBackend{
		entries: make(map[string]time.Time),
	}
}

func (r *SimpleRouteBackend) Name() string { return "simple_route" }

// MarkIP adds a route with expires based on DNS TTL
func (r *SimpleRouteBackend) MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Validate inputs
	if err := r.validateInputs(iface, ip); err != nil {
		return err
	}

	normalizedIP, _ := NormalizeIP(ip)
	ttlSeconds = r.normalizeTTL(ttlSeconds)

	// Check if route already exists with longer TTL
	if r.shouldSkipRoute(normalizedIP, ttlSeconds) {
		zerolog.Ctx(ctx).Debug().
			IPAddr("ip", net.ParseIP(normalizedIP)).
			Str("iface", iface).
			Int("ttl", ttlSeconds).
			Msg("route already exists with longer TTL, skipping")
		return nil
	}

	// Add route with expires
	if err := r.addRouteWithExpires(ctx, normalizedIP, iface, ttlSeconds); err != nil {
		return fmt.Errorf("failed to add route: %w", err)
	}

	// Track expiry time
	r.entries[normalizedIP] = time.Now().Add(time.Duration(ttlSeconds) * time.Second)

	zerolog.Ctx(ctx).Debug().
		IPAddr("ip", net.ParseIP(normalizedIP)).
		Str("iface", iface).
		Int("ttl", ttlSeconds).
		Msg("route added with expires")

	return nil
}

// CleanupAll removes all tracked entries (routes will expire automatically)
func (r *SimpleRouteBackend) CleanupAll(ctx context.Context) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	log := zerolog.Ctx(ctx)
	log.Info().Msg("clearing route tracking (routes will expire automatically)")

	// Clear tracking - actual routes will expire automatically
	r.entries = make(map[string]time.Time)

	return nil
}

// addRouteWithExpires adds a route with expires parameter
func (r *SimpleRouteBackend) addRouteWithExpires(ctx context.Context, ip, iface string, ttlSeconds int) error {
	// Try to add route with expires
	cmd := exec.CommandContext(ctx, "ip", "route", "add",
		ip+"/32", "dev", iface, "proto", "186", "scope", "link", "expires", strconv.Itoa(ttlSeconds))

	out, err := cmd.CombinedOutput()
	if err != nil {
		output := string(out)

		// Check if route already exists
		if strings.Contains(output, "already exists") {
			// Try to update existing route with new expires
			return r.updateRouteExpires(ctx, ip, iface, ttlSeconds)
		}

		return fmt.Errorf("failed to add route: %s", output)
	}

	return nil
}

// updateRouteExpires updates the expires time of an existing route
func (r *SimpleRouteBackend) updateRouteExpires(ctx context.Context, ip, iface string, ttlSeconds int) error {
	// Delete existing route
	delCmd := exec.CommandContext(ctx, "ip", "route", "del", ip+"/32", "dev", iface)
	if out, err := delCmd.CombinedOutput(); err != nil {
		// Route might not exist, continue anyway
		zerolog.Ctx(ctx).Debug().Bytes("out", out).Msg("route delete failed (may not exist)")
	}

	// Add route with new expires
	addCmd := exec.CommandContext(ctx, "ip", "route", "add",
		ip+"/32", "dev", iface, "proto", "186", "scope", "link", "expires", strconv.Itoa(ttlSeconds))

	if out, err := addCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update route: %s", string(out))
	}

	zerolog.Ctx(ctx).Debug().
		Str("ip", ip).
		Str("iface", iface).
		Int("ttl", ttlSeconds).
		Msg("route updated with new expires")

	return nil
}

// shouldSkipRoute checks if we should skip adding a route
func (r *SimpleRouteBackend) shouldSkipRoute(normalizedIP string, ttlSeconds int) bool {
	existing, exists := r.entries[normalizedIP]
	if !exists {
		return false
	}

	// Skip if existing route expires later than new TTL
	remainingTime := time.Until(existing)
	return remainingTime > time.Duration(ttlSeconds)*time.Second
}

// validateInputs validates interface and IP inputs
func (r *SimpleRouteBackend) validateInputs(iface, ip string) error {
	if !IsSafeIfaceName(iface) {
		return fmt.Errorf("%w: %q", ErrInvalidIface, iface)
	}

	if _, ok := NormalizeIP(ip); !ok {
		return fmt.Errorf("%w: %q", ErrInvalidIP, ip)
	}

	return nil
}

// normalizeTTL ensures TTL is within reasonable bounds
func (r *SimpleRouteBackend) normalizeTTL(ttlSeconds int) int {
	const maxTTL = 3600 // Maximum 1 hour
	return min(max(ttlSeconds, minTTLSeconds), maxTTL)
}
