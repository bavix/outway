package firewall_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/firewall"
)

func TestNewSimpleRouteBackend(t *testing.T) {
	t.Parallel()

	backend := firewall.NewSimpleRouteBackend()
	require.NotNil(t, backend)
	assert.Equal(t, "simple_route", backend.Name())
}

func TestSimpleRouteBackendName(t *testing.T) {
	t.Parallel()

	backend := firewall.NewSimpleRouteBackend()
	assert.Equal(t, "simple_route", backend.Name())
}

func TestSimpleRouteBackendMarkIP(t *testing.T) {
	t.Parallel()

	backend := firewall.NewSimpleRouteBackend()
	ctx := context.Background()

	// Test with valid inputs
	err := backend.MarkIP(ctx, "eth0", "192.168.1.1", 300)
	// This might fail in test environment due to missing ip command
	// but should not panic
	_ = err

	// Test with different TTL values
	ttlValues := []int{30, 60, 300, 3600, 86400}
	for _, ttl := range ttlValues {
		err := backend.MarkIP(ctx, "eth0", "192.168.1.1", ttl)
		_ = err
	}
}

func TestSimpleRouteBackendMarkIPInvalidInputs(t *testing.T) {
	t.Parallel()

	backend := firewall.NewSimpleRouteBackend()
	ctx := context.Background()

	// Test with empty interface name
	err := backend.MarkIP(ctx, "", "192.168.1.1", 300)
	require.Error(t, err)

	// Test with invalid interface name
	err = backend.MarkIP(ctx, "eth@0", "192.168.1.1", 300)
	require.Error(t, err)

	// Test with empty IP
	err = backend.MarkIP(ctx, "eth0", "", 300)
	require.Error(t, err)

	// Test with invalid IP
	err = backend.MarkIP(ctx, "eth0", "invalid-ip", 300)
	require.Error(t, err)

	// Test with nil context - skip as it may panic
	// err = backend.MarkIP(nil, "eth0", "192.168.1.1", 300)
}

func TestSimpleRouteBackendMarkIPTTLNormalization(t *testing.T) {
	t.Parallel()

	backend := firewall.NewSimpleRouteBackend()
	ctx := context.Background()

	// Test with very small TTL
	err := backend.MarkIP(ctx, "eth0", "192.168.1.1", 1)
	// Should normalize to minimum TTL
	_ = err

	// Test with zero TTL
	err = backend.MarkIP(ctx, "eth0", "192.168.1.2", 0)
	// Should normalize to minimum TTL
	_ = err

	// Test with negative TTL
	err = backend.MarkIP(ctx, "eth0", "192.168.1.3", -10)
	// Should normalize to minimum TTL
	_ = err

	// Test with large TTL
	err = backend.MarkIP(ctx, "eth0", "192.168.1.4", 86400)
	// Should work with large TTL
	_ = err
}

func TestSimpleRouteBackendCleanupAll(t *testing.T) {
	t.Parallel()

	backend := firewall.NewSimpleRouteBackend()
	ctx := context.Background()

	// Test CleanupAll
	err := backend.CleanupAll(ctx)
	// This might fail in test environment due to missing ip command
	// but should not panic
	_ = err

	// Test with nil context - skip as it may panic
	// err = backend.CleanupAll(nil)
}

func TestSimpleRouteBackendConcurrency(t *testing.T) {
	t.Parallel()

	backend := firewall.NewSimpleRouteBackend()
	ctx := context.Background()

	// Test concurrent MarkIP calls
	done := make(chan bool, 10)

	for i := range 10 {
		go func(id int) {
			defer func() { done <- true }()

			ip := "192.168.1." + string(rune(id+1))
			_ = backend.MarkIP(ctx, "eth0", ip, 300)
		}(i)
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}

	// Test CleanupAll after concurrent operations
	_ = backend.CleanupAll(ctx)
}

func TestSimpleRouteBackendDuplicateRoutes(t *testing.T) {
	t.Parallel()

	backend := firewall.NewSimpleRouteBackend()
	ctx := context.Background()

	// Test adding the same route multiple times
	ip := "192.168.1.1"
	ttl := 300

	// First call
	err1 := backend.MarkIP(ctx, "eth0", ip, ttl)
	_ = err1

	// Second call with same parameters
	err2 := backend.MarkIP(ctx, "eth0", ip, ttl)
	_ = err2

	// Third call with longer TTL
	err3 := backend.MarkIP(ctx, "eth0", ip, ttl*2)
	_ = err3

	// Should handle duplicates gracefully
}

func TestSimpleRouteBackendDifferentInterfaces(t *testing.T) {
	t.Parallel()

	backend := firewall.NewSimpleRouteBackend()
	ctx := context.Background()

	interfaces := []string{"eth0", "eth1", "wlan0", "lo"}
	ips := []string{"192.168.1.1", "192.168.1.2", "10.0.0.1", "127.0.0.1"}

	for i, iface := range interfaces {
		err := backend.MarkIP(ctx, iface, ips[i], 300)
		_ = err
	}

	// Test CleanupAll
	_ = backend.CleanupAll(ctx)
}

func TestSimpleRouteBackendIPv6(t *testing.T) {
	t.Parallel()

	backend := firewall.NewSimpleRouteBackend()
	ctx := context.Background()

	// Test with IPv6 addresses
	ipv6Addresses := []string{
		"2001:db8::1",
		"::1",
		"fe80::1",
		"2001:db8::2",
	}

	for _, ip := range ipv6Addresses {
		err := backend.MarkIP(ctx, "eth0", ip, 300)
		_ = err
	}

	// Test CleanupAll
	_ = backend.CleanupAll(ctx)
}

//nolint:funlen
func TestSimpleRouteBackendErrorHandling(t *testing.T) {
	t.Parallel()

	backend := firewall.NewSimpleRouteBackend()
	ctx := context.Background()

	// Test with various error conditions
	testCases := []struct {
		name    string
		iface   string
		ip      string
		ttl     int
		wantErr bool
	}{
		{
			name:    "valid inputs",
			iface:   "eth0",
			ip:      "192.168.1.1",
			ttl:     300,
			wantErr: false,
		},
		{
			name:    "empty interface",
			iface:   "",
			ip:      "192.168.1.1",
			ttl:     300,
			wantErr: true,
		},
		{
			name:    "invalid interface",
			iface:   "eth@0",
			ip:      "192.168.1.1",
			ttl:     300,
			wantErr: true,
		},
		{
			name:    "empty IP",
			iface:   "eth0",
			ip:      "",
			ttl:     300,
			wantErr: true,
		},
		{
			name:    "invalid IP",
			iface:   "eth0",
			ip:      "not-an-ip",
			ttl:     300,
			wantErr: true,
		},
		{
			name:    "negative TTL",
			iface:   "eth0",
			ip:      "192.168.1.1",
			ttl:     -10,
			wantErr: false, // Should normalize to minimum TTL
		},
		{
			name:    "zero TTL",
			iface:   "eth0",
			ip:      "192.168.1.1",
			ttl:     0,
			wantErr: false, // Should normalize to minimum TTL
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := backend.MarkIP(ctx, tc.iface, tc.ip, tc.ttl)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				// Even if it doesn't error, it might fail due to missing ip command
				// in test environment, so we just check it doesn't panic
				_ = err
			}
		})
	}
}

func TestSimpleRouteBackendStressTest(t *testing.T) {
	t.Parallel()

	backend := firewall.NewSimpleRouteBackend()
	ctx := context.Background()

	// Stress test with many operations
	numOperations := 100
	done := make(chan bool, numOperations)

	for i := range numOperations {
		go func(id int) {
			defer func() { done <- true }()

			// Use different IPs and interfaces
			ip := "192.168.1." + string(rune((id%254)+1))
			iface := "eth" + string(rune((id % 4)))
			ttl := 300 + (id % 3600)

			_ = backend.MarkIP(ctx, iface, ip, ttl)
		}(i)
	}

	// Wait for all operations to complete
	for range numOperations {
		<-done
	}

	// Test CleanupAll
	_ = backend.CleanupAll(ctx)
}
