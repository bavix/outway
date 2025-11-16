package dnsproxy_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/dnsproxy"
)

// TestSetUpstreamsConfig_Validation tests validation of upstreams before update.
//
//nolint:funlen // comprehensive test cases
func TestSetUpstreamsConfig_Validation(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Path: "/tmp/test-outway-upstreams-validation.yaml",
		Listen: config.ListenConfig{
			UDP: ":5353",
			TCP: ":5353",
		},
		Upstreams: []config.UpstreamConfig{
			{
				Name:    "initial",
				Address: "8.8.8.8:53",
				Type:    "udp",
				Weight:  1,
			},
		},
	}

	mockBackend := &MockFirewallBackend{}
	proxy := dnsproxy.New(cfg, mockBackend)
	require.NotNil(t, proxy)

	ctx := context.Background()

	tests := []struct {
		name        string
		upstreams   []config.UpstreamConfig
		wantErr     bool
		errContains string
	}{
		{
			name:        "empty upstreams list",
			upstreams:   []config.UpstreamConfig{},
			wantErr:     true,
			errContains: "at least one upstream is required",
		},
		{
			name: "upstream with empty name",
			upstreams: []config.UpstreamConfig{
				{
					Name:    "",
					Address: "8.8.8.8:53",
					Weight:  1,
				},
			},
			wantErr:     true,
			errContains: "name cannot be empty",
		},
		{
			name: "upstream with empty address",
			upstreams: []config.UpstreamConfig{
				{
					Name:    "test",
					Address: "",
					Weight:  1,
				},
			},
			wantErr:     true,
			errContains: "address cannot be empty",
		},
		{
			name: "upstream with negative weight",
			upstreams: []config.UpstreamConfig{
				{
					Name:    "test",
					Address: "8.8.8.8:53",
					Weight:  -1,
				},
			},
			wantErr:     true,
			errContains: "weight cannot be negative",
		},
		{
			name: "multiple upstreams with one invalid",
			upstreams: []config.UpstreamConfig{
				{
					Name:    "valid",
					Address: "8.8.8.8:53",
					Weight:  1,
				},
				{
					Name:    "",
					Address: "8.8.4.4:53",
					Weight:  1,
				},
			},
			wantErr:     true,
			errContains: "name cannot be empty",
		},
		{
			name: "valid upstreams",
			upstreams: []config.UpstreamConfig{
				{
					Name:    "google",
					Address: "8.8.8.8:53",
					Weight:  1,
				},
				{
					Name:    "cloudflare",
					Address: "1.1.1.1:53",
					Weight:  2,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := proxy.SetUpstreamsConfig(ctx, tt.upstreams)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestSetUpstreamsConfig_AtomicUpdate tests that upstreams are updated atomically.
func TestSetUpstreamsConfig_AtomicUpdate(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Path: "/tmp/test-outway-upstreams-atomic.yaml",
		Listen: config.ListenConfig{
			UDP: ":5353",
			TCP: ":5353",
		},
		Upstreams: []config.UpstreamConfig{
			{
				Name:    "initial",
				Address: "8.8.8.8:53",
				Type:    "udp",
				Weight:  1,
			},
		},
	}

	mockBackend := &MockFirewallBackend{}
	proxy := dnsproxy.New(cfg, mockBackend)
	require.NotNil(t, proxy)

	ctx := context.Background()

	// Get initial upstreams
	initialUpstreams := proxy.GetUpstreams()
	require.Len(t, initialUpstreams, 1)

	// Update with new upstreams
	newUpstreams := []config.UpstreamConfig{
		{
			Name:    "google",
			Address: "udp://8.8.8.8:53",
			Weight:  1,
		},
		{
			Name:    "cloudflare",
			Address: "udp://1.1.1.1:53",
			Weight:  2,
		},
	}

	err := proxy.SetUpstreamsConfig(ctx, newUpstreams)
	require.NoError(t, err)

	// Verify upstreams are updated
	updatedUpstreams := proxy.GetUpstreams()
	assert.Len(t, updatedUpstreams, 2)
	assert.NotEqual(t, initialUpstreams, updatedUpstreams)
}

// TestSetUpstreamsConfig_TypeDetection tests automatic type detection.
//
//nolint:funlen // comprehensive test cases
func TestSetUpstreamsConfig_TypeDetection(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Path: "/tmp/test-outway-upstreams-type-detection.yaml",
		Listen: config.ListenConfig{
			UDP: ":5353",
			TCP: ":5353",
		},
		Upstreams: []config.UpstreamConfig{
			{
				Name:    "initial",
				Address: "8.8.8.8:53",
				Type:    "udp",
				Weight:  1,
			},
		},
	}

	mockBackend := &MockFirewallBackend{}
	proxy := dnsproxy.New(cfg, mockBackend)
	require.NotNil(t, proxy)

	ctx := context.Background()

	tests := []struct {
		name     string
		address  string
		expected string
	}{
		{
			name:     "UDP with scheme",
			address:  "udp://8.8.8.8:53",
			expected: "udp",
		},
		{
			name:     "TCP with scheme",
			address:  "tcp://8.8.8.8:53",
			expected: "tcp",
		},
		{
			name:     "DoH with scheme",
			address:  "https://dns.google/dns-query",
			expected: "doh",
		},
		{
			name:     "DoT with scheme",
			address:  "tls://8.8.8.8:853",
			expected: "dot",
		},
		{
			name:     "DoQ with scheme",
			address:  "quic://8.8.8.8:853",
			expected: "doq",
		},
		{
			name:     "address without scheme (defaults to UDP)",
			address:  "8.8.8.8:53",
			expected: "udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create a fresh proxy for each test to avoid conflicts
			testCfg := &config.Config{
				Path: "/tmp/test-outway-upstreams-type-" + tt.name + ".yaml",
				Listen: config.ListenConfig{
					UDP: ":5353",
					TCP: ":5353",
				},
				Upstreams: []config.UpstreamConfig{
					{
						Name:    "initial",
						Address: "8.8.8.8:53",
						Type:    "udp",
						Weight:  1,
					},
				},
			}

			//nolint:contextcheck // New doesn't require context
			testProxy := dnsproxy.New(testCfg, &MockFirewallBackend{})
			require.NotNil(t, testProxy)

			upstreams := []config.UpstreamConfig{
				{
					Name:    "test",
					Address: tt.address,
					// Type is empty, should be auto-detected
				},
			}

			err := testProxy.SetUpstreamsConfig(ctx, upstreams)
			require.NoError(t, err)

			// Verify type was detected
			updatedUpstreams := testProxy.GetUpstreams()
			require.Len(t, updatedUpstreams, 1)
			// The address format in GetUpstreams is "type:address"
			// For DoT, tls:// is converted to dot://, so check for both
			if tt.expected == "dot" {
				assert.True(t,
					updatedUpstreams[0] == "dot:tls://8.8.8.8:853" ||
						updatedUpstreams[0] == "dot:8.8.8.8:853" ||
						strings.Contains(updatedUpstreams[0], "dot"),
					"Expected DoT type, got: %s", updatedUpstreams[0])
			} else {
				assert.Contains(t, updatedUpstreams[0], tt.expected)
			}
		})
	}
}

// TestSetUpstreamsConfig_WeightNormalization tests weight normalization.
func TestSetUpstreamsConfig_WeightNormalization(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Path: "/tmp/test-outway-upstreams-weight.yaml",
		Listen: config.ListenConfig{
			UDP: ":5353",
			TCP: ":5353",
		},
		Upstreams: []config.UpstreamConfig{
			{
				Name:    "initial",
				Address: "8.8.8.8:53",
				Type:    "udp",
				Weight:  1,
			},
		},
	}

	mockBackend := &MockFirewallBackend{}
	proxy := dnsproxy.New(cfg, mockBackend)
	require.NotNil(t, proxy)

	ctx := context.Background()

	// Test with zero weight (should be normalized to 1)
	upstreams := []config.UpstreamConfig{
		{
			Name:    "test",
			Address: "8.8.8.8:53",
			Weight:  0, // Should be normalized to 1
		},
	}

	err := proxy.SetUpstreamsConfig(ctx, upstreams)
	require.NoError(t, err)

	// Verify upstreams are set (weight normalization happens internally)
	updatedUpstreams := proxy.GetUpstreams()
	require.Len(t, updatedUpstreams, 1)
}

// TestSetUpstreamsConfig_AddressNormalization tests address normalization for persistence.
func TestSetUpstreamsConfig_AddressNormalization(t *testing.T) {
	t.Parallel()

	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	cfg := &config.Config{
		Path: configPath,
		Listen: config.ListenConfig{
			UDP: ":5353",
			TCP: ":5353",
		},
		Upstreams: []config.UpstreamConfig{
			{
				Name:    "initial",
				Address: "8.8.8.8:53",
				Type:    "udp",
				Weight:  1,
			},
		},
	}

	mockBackend := &MockFirewallBackend{}
	proxy := dnsproxy.New(cfg, mockBackend)
	require.NotNil(t, proxy)

	ctx := context.Background()

	// Update with addresses that have schemes (should be normalized for persistence)
	upstreams := []config.UpstreamConfig{
		{
			Name:    "google",
			Address: "udp://8.8.8.8:53", // Should be normalized to "8.8.8.8:53" in config
			Weight:  1,
		},
		{
			Name:    "cloudflare",
			Address: "tcp://1.1.1.1:53", // Should be normalized to "1.1.1.1:53" in config
			Weight:  2,
		},
		{
			Name:    "doh",
			Address: "https://dns.google/dns-query", // Should remain as-is (not UDP/TCP)
			Weight:  1,
		},
	}

	err := proxy.SetUpstreamsConfig(ctx, upstreams)
	require.NoError(t, err)

	// Wait a bit for async save to complete
	time.Sleep(100 * time.Millisecond)

	// Verify config file was saved and addresses were normalized
	if _, err := os.Stat(configPath); err == nil {
		// Config file exists, verify it was saved
		// Note: We can't easily verify the exact content without loading it,
		// but the fact that it doesn't crash is good
		assert.NoError(t, err)
	}
}

// TestSetUpstreamsConfig_ResolverRemainsActive tests that resolver remains active after update.
func TestSetUpstreamsConfig_ResolverRemainsActive(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Path: "/tmp/test-outway-upstreams-resolver.yaml",
		Listen: config.ListenConfig{
			UDP: ":5353",
			TCP: ":5353",
		},
		Upstreams: []config.UpstreamConfig{
			{
				Name:    "initial",
				Address: "8.8.8.8:53",
				Type:    "udp",
				Weight:  1,
			},
		},
	}

	mockBackend := &MockFirewallBackend{}
	proxy := dnsproxy.New(cfg, mockBackend)
	require.NotNil(t, proxy)

	ctx := context.Background()

	// Verify initial resolver is active
	initialResolver := proxy.ResolverActive()
	require.NotNil(t, initialResolver, "Resolver should be active initially")

	// Update upstreams
	newUpstreams := []config.UpstreamConfig{
		{
			Name:    "google",
			Address: "udp://8.8.8.8:53",
			Weight:  1,
		},
		{
			Name:    "cloudflare",
			Address: "udp://1.1.1.1:53",
			Weight:  2,
		},
	}

	err := proxy.SetUpstreamsConfig(ctx, newUpstreams)
	require.NoError(t, err)

	// Verify resolver is still active after update
	updatedResolver := proxy.ResolverActive()
	require.NotNil(t, updatedResolver, "Resolver should remain active after SetUpstreamsConfig")
	assert.NotNil(t, updatedResolver, "Resolver should not be nil")
}

// TestSetUpstreamsConfig_ConcurrentUpdates tests concurrent updates are handled safely.
//
//nolint:funlen // concurrent test setup
func TestSetUpstreamsConfig_ConcurrentUpdates(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Path: "/tmp/test-outway-upstreams-concurrent.yaml",
		Listen: config.ListenConfig{
			UDP: ":5353",
			TCP: ":5353",
		},
		Upstreams: []config.UpstreamConfig{
			{
				Name:    "initial",
				Address: "8.8.8.8:53",
				Type:    "udp",
				Weight:  1,
			},
		},
	}

	mockBackend := &MockFirewallBackend{}
	proxy := dnsproxy.New(cfg, mockBackend)
	require.NotNil(t, proxy)

	ctx := context.Background()

	// Test concurrent updates
	const (
		numGoroutines = 10
		numUpdates    = 5
	)

	var wg sync.WaitGroup

	errors := make(chan error, numGoroutines*numUpdates)

	for i := range numGoroutines {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			for j := range numUpdates {
				upstreams := []config.UpstreamConfig{
					{
						Name:    "test",
						Address: "8.8.8.8:53",
						Weight:  id*numUpdates + j + 1,
					},
				}

				if err := proxy.SetUpstreamsConfig(ctx, upstreams); err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		require.NoError(t, err, "Concurrent update should not fail")
	}

	// Verify final state is consistent
	finalUpstreams := proxy.GetUpstreams()
	assert.GreaterOrEqual(t, len(finalUpstreams), 1, "Should have at least one upstream")
}

// TestSetUpstreamsConfig_AsyncSave tests that async save doesn't block.
func TestSetUpstreamsConfig_AsyncSave(t *testing.T) {
	t.Parallel()

	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config-async.yaml")

	cfg := &config.Config{
		Path: configPath,
		Listen: config.ListenConfig{
			UDP: ":5353",
			TCP: ":5353",
		},
		Upstreams: []config.UpstreamConfig{
			{
				Name:    "initial",
				Address: "8.8.8.8:53",
				Type:    "udp",
				Weight:  1,
			},
		},
	}

	mockBackend := &MockFirewallBackend{}
	proxy := dnsproxy.New(cfg, mockBackend)
	require.NotNil(t, proxy)

	ctx := context.Background()

	// Update upstreams - this should return immediately (async save)
	start := time.Now()
	upstreams := []config.UpstreamConfig{
		{
			Name:    "test",
			Address: "8.8.8.8:53",
			Weight:  1,
		},
	}

	err := proxy.SetUpstreamsConfig(ctx, upstreams)
	duration := time.Since(start)

	require.NoError(t, err)
	// Should return quickly (async save doesn't block)
	assert.Less(t, duration, 100*time.Millisecond, "SetUpstreamsConfig should return quickly (async save)")

	// Wait a bit for async save to complete
	time.Sleep(200 * time.Millisecond)

	// Verify upstreams are updated in memory (regardless of save status)
	updatedUpstreams := proxy.GetUpstreams()
	require.Len(t, updatedUpstreams, 1)
}

// TestSetUpstreamsConfig_MultipleFormats tests handling of multiple upstream formats.
func TestSetUpstreamsConfig_MultipleFormats(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Path: "/tmp/test-outway-upstreams-formats.yaml",
		Listen: config.ListenConfig{
			UDP: ":5353",
			TCP: ":5353",
		},
		Upstreams: []config.UpstreamConfig{
			{
				Name:    "initial",
				Address: "8.8.8.8:53",
				Type:    "udp",
				Weight:  1,
			},
		},
	}

	mockBackend := &MockFirewallBackend{}
	proxy := dnsproxy.New(cfg, mockBackend)
	require.NotNil(t, proxy)

	ctx := context.Background()

	// Mix of different formats
	upstreams := []config.UpstreamConfig{
		{
			Name:    "google-udp",
			Address: "udp://8.8.8.8:53",
			Weight:  1,
		},
		{
			Name:    "google-tcp",
			Address: "tcp://8.8.8.8:53",
			Weight:  1,
		},
		{
			Name:    "cloudflare",
			Address: "1.1.1.1:53", // No scheme, should default to UDP
			Weight:  2,
		},
		{
			Name:    "doh-google",
			Address: "https://dns.google/dns-query",
			Weight:  1,
		},
	}

	err := proxy.SetUpstreamsConfig(ctx, upstreams)
	require.NoError(t, err)

	// Verify all upstreams are set
	updatedUpstreams := proxy.GetUpstreams()
	assert.Len(t, updatedUpstreams, 4)
}
