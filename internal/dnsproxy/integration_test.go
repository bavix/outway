package dnsproxy

import (
	"context"
	"os"
	"testing"

	"github.com/bavix/outway/internal/config"
)

// TestHostsUpdateDoesNotBreakUpstreamResolution is a comprehensive integration test
// that reproduces the exact bug scenario: updating hosts should not break upstream resolution
func TestHostsUpdateDoesNotBreakUpstreamResolution(t *testing.T) {
	// Create a config file similar to what users would have
	testConfig := `app_name: outway
listen:
  udp: :5353
  tcp: :5353
upstreams:
- name: cloudflare
  address: 1.1.1.1:53
  weight: 1
- name: google
  address: 8.8.8.8:53
  weight: 1
rule_groups:
- name: default
  via: default
  patterns:
  - "*"
cache:
  enabled: true
  max_entries: 1000
hosts:
- pattern: initial.test
  a:
  - 127.0.0.1
  ttl: 60
`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testConfig)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// Load config
	cfg, err := config.Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create proxy
	backend := &nullBackend{}
	proxy := New(cfg, backend)
	ctx := context.Background()

	// Initialize resolver
	proxy.rebuildResolver(ctx)

	// Verify initial state - upstreams should be configured
	resolver := proxy.ResolverActive()
	if resolver == nil {
		t.Fatal("Resolver is nil after initialization")
	}
	t.Log("Initial resolver is active")

	// Get initial upstreams
	upstreamsBefore := proxy.GetUpstreams()
	t.Logf("Upstreams before SetHosts: %v", upstreamsBefore)
	if len(upstreamsBefore) == 0 {
		t.Fatal("No upstreams found initially")
	}

	// Update hosts via SetHosts (this is what the admin panel does)
	newHosts := []config.HostOverride{
		{
			Pattern: "updated.test",
			A:       []string{"127.0.0.2"},
			TTL:     60,
		},
		{
			Pattern: "another.test",
			A:       []string{"127.0.0.3"},
			TTL:     120,
		},
	}

	if err := proxy.SetHosts(ctx, newHosts); err != nil {
		t.Fatalf("SetHosts failed: %v", err)
	}

	// Verify resolver is still active after SetHosts
	resolver = proxy.ResolverActive()
	if resolver == nil {
		t.Fatal("BUG: Resolver became nil after SetHosts!")
	}
	t.Log("Resolver is still active after SetHosts")

	// Verify upstreams still work for non-host queries
	// This is the critical test - upstream resolution should NOT be broken
	upstreamsAfter := proxy.GetUpstreams()
	t.Logf("Upstreams after SetHosts: %v", upstreamsAfter)
	
	if len(upstreamsAfter) == 0 {
		t.Fatal("BUG: All upstreams were lost after SetHosts! This is the bug we're fixing.")
	}

	if len(upstreamsAfter) != len(upstreamsBefore) {
		t.Errorf("Number of upstreams changed: before=%d, after=%d", len(upstreamsBefore), len(upstreamsAfter))
	}

	// Verify upstreams in config still have types
	for _, u := range cfg.Upstreams {
		if u.Type == "" {
			t.Errorf("BUG: Upstream %s has empty type after SetHosts", u.Name)
		}
		t.Logf("Upstream %s: address=%s, type=%s, weight=%d", u.Name, u.Address, u.Type, u.Weight)
	}

	// Verify config can be reloaded without losing upstream types
	cfg2, err := config.Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	for _, u := range cfg2.Upstreams {
		if u.Type == "" {
			t.Errorf("BUG: Upstream %s has empty type after config reload", u.Name)
		}
	}

	t.Log("SUCCESS: Hosts update did not break upstream resolution!")
}

// TestMultipleHostsUpdates tests that multiple consecutive hosts updates don't break anything
func TestMultipleHostsUpdates(t *testing.T) {
	testConfig := `app_name: outway
listen:
  udp: :5353
  tcp: :5353
upstreams:
- name: cloudflare
  address: 1.1.1.1:53
  weight: 1
rule_groups:
- name: default
  via: default
  patterns:
  - "*"
`

	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(testConfig)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg, err := config.Load(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	backend := &nullBackend{}
	proxy := New(cfg, backend)
	ctx := context.Background()
	proxy.rebuildResolver(ctx)

	// Perform multiple hosts updates
	for i := 0; i < 5; i++ {
		hosts := []config.HostOverride{
			{
				Pattern: "test.local",
				A:       []string{"127.0.0.1"},
				TTL:     60,
			},
		}

		if err := proxy.SetHosts(ctx, hosts); err != nil {
			t.Fatalf("SetHosts #%d failed: %v", i+1, err)
		}

		// Verify upstreams are still there
		upstreams := proxy.GetUpstreams()
		if len(upstreams) == 0 {
			t.Fatalf("BUG: Upstreams lost after update #%d", i+1)
		}

		// Verify resolver is still active
		if proxy.ResolverActive() == nil {
			t.Fatalf("BUG: Resolver became nil after update #%d", i+1)
		}
	}

	t.Log("SUCCESS: Multiple hosts updates did not break upstream resolution!")
}
