package dnsproxy_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/dnsproxy"
	"github.com/bavix/outway/internal/firewall"
)

// MockFirewallBackend is a mock implementation of firewall.Backend for testing.
type MockFirewallBackend struct{}

func (m *MockFirewallBackend) Name() string {
	return "mock"
}

func (m *MockFirewallBackend) MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error {
	return nil
}

func (m *MockFirewallBackend) CleanupAll(ctx context.Context) error {
	return nil
}

// Ensure MockFirewallBackend implements firewall.Backend interface.
var _ firewall.Backend = (*MockFirewallBackend)(nil)

// TestSetHostsIntegration tests the integration of SetHosts with dynamic HostsResolver.
//
//nolint:funlen // comprehensive integration test
func TestSetHostsIntegration(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Path: "/tmp/test-outway-config.yaml", // Set path to avoid save errors
		Listen: config.ListenConfig{
			UDP: ":5353",
			TCP: ":5353",
		},
		History: config.HistoryConfig{
			MaxEntries: 100,
		},
		Upstreams: []config.UpstreamConfig{
			{
				Name:    "test-upstream",
				Address: "8.8.8.8:53",
				Type:    "udp",
				Weight:  1,
			},
		},
		Cache: config.CacheConfig{
			Enabled: true,
		},
	}

	mockBackend := &MockFirewallBackend{}
	proxy := dnsproxy.New(cfg, mockBackend)

	require.NotNil(t, proxy)
	require.NotNil(t, proxy.ResolverActive())

	ctx := context.Background()

	// Test 1: Set initial hosts
	initialHosts := []config.HostOverride{
		{
			Pattern: "test.local",
			A:       []string{"192.168.1.100"},
			TTL:     60,
		},
	}

	err := proxy.SetHosts(ctx, initialHosts)
	require.NoError(t, err)

	// Verify hosts are set
	hosts := proxy.GetHosts()
	assert.Len(t, hosts, 1)
	assert.Equal(t, "test.local", hosts[0].Pattern)

	// Test 2: Verify resolver is active (SetHosts should not break resolver)
	resolver := proxy.ResolverActive()
	require.NotNil(t, resolver, "Resolver should be active after SetHosts")

	// Note: We don't test actual resolution here as it requires network access
	// The important part is that SetHosts doesn't crash and resolver remains active

	// Test 3: Update hosts
	updatedHosts := []config.HostOverride{
		{
			Pattern: "test.local",
			A:       []string{"192.168.1.200"},
			TTL:     120,
		},
		{
			Pattern: "another.local",
			A:       []string{"192.168.1.201"},
			TTL:     60,
		},
	}

	err = proxy.SetHosts(ctx, updatedHosts)
	require.NoError(t, err)

	// Verify hosts are updated
	hosts = proxy.GetHosts()
	assert.Len(t, hosts, 2)
	assert.Equal(t, "test.local", hosts[0].Pattern)
	assert.Equal(t, "another.local", hosts[1].Pattern)
}

// TestUpstreamFormatsIntegration tests that proxy can be created with different upstream formats.
func TestUpstreamFormatsIntegration(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		address string
	}{
		{
			name:    "UDP with scheme",
			address: "udp://8.8.8.8:53",
		},
		{
			name:    "TCP with scheme",
			address: "tcp://8.8.8.8:53",
		},
		{
			name:    "UDP without scheme",
			address: "8.8.8.8:53",
		},
		{
			name:    "IPv6 UDP with scheme",
			address: "udp://[2606:4700:4700::1111]:53",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.Config{
				Listen: config.ListenConfig{
					UDP: ":5353",
					TCP: ":5353",
				},
				Upstreams: []config.UpstreamConfig{
					{
						Name:    "test",
						Address: tc.address,
						Weight:  1,
					},
				},
			}

			mockBackend := &MockFirewallBackend{}
			proxy := dnsproxy.New(cfg, mockBackend)

			// Proxy should be created successfully regardless of upstream format
			assert.NotNil(t, proxy)
			assert.NotNil(t, proxy.ResolverActive())
		})
	}
}

// TestCacheMemoryLimitIntegration tests that cache respects MaxSizeMB limit.
func TestCacheMemoryLimitIntegration(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Cache: config.CacheConfig{
			Enabled:    true,
			MaxEntries: 1000,
			MaxSizeMB:  1, // 1MB limit
		},
	}

	// Create a cache resolver
	next := &MockResolver{}
	cache := dnsproxy.NewCachedResolverWithSize(next, cfg.Cache.MaxEntries, cfg.Cache.MaxSizeMB, 60, 3600)

	require.NotNil(t, cache)
	assert.Equal(t, int64(1024*1024), cache.MaxSizeBytes, "MaxSizeBytes should be 1MB")
}
