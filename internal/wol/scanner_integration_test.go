package wol_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/wol"
)

// TestNetworkScanner_Integration tests real network scanning.
func TestNetworkScanner_Integration(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	scanner := wol.NewNetworkScanner()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get local network
	networkCIDR, err := scanner.GetLocalNetworkCIDR()
	require.NoError(t, err)

	t.Logf("Scanning network: %s", networkCIDR)

	// Perform scan
	results, err := scanner.ScanNetwork(ctx, networkCIDR)

	// Should not error
	require.NoError(t, err)
	assert.NotNil(t, results)

	t.Logf("Found %d devices", len(results))

	// Log found devices
	for i, result := range results {
		t.Logf("Device %d: IP=%s, MAC=%s, Status=%s",
			i+1, result.IP, result.MAC, result.Status)
	}

	// Should find at least some devices (router, etc.)
	// But we can't guarantee this in all environments
	if len(results) > 0 {
		// Check that all results have valid IPs
		for _, result := range results {
			assert.NotEmpty(t, result.IP)
			assert.Equal(t, "online", result.Status)
		}
	}
}

// TestNetworkScanner_RealScan tests with a real small network.
func TestNetworkScanner_RealScan(t *testing.T) {
	t.Parallel()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	scanner := wol.NewNetworkScanner()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with localhost network (localhost might not respond to ping on macOS)
	results, err := scanner.ScanNetwork(ctx, "127.0.0.0/30")

	require.NoError(t, err)

	t.Logf("Localhost scan found %d devices", len(results))

	// On macOS, localhost might not respond to ping, so we just check that scan completes
	// without error and returns a valid (possibly empty) result
}
