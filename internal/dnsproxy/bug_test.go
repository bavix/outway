package dnsproxy

import (
	"context"
	"os"
	"testing"

	"github.com/bavix/outway/internal/config"
)

// nullBackend is a simple test backend that does nothing
type nullBackend struct{}

func (n *nullBackend) Name() string                                                    { return "null" }
func (n *nullBackend) MarkIP(ctx context.Context, iface, ip string, ttlSeconds int) error { return nil }
func (n *nullBackend) CleanupAll(ctx context.Context) error                           { return nil }

// Test to reproduce the bug where saving hosts breaks upstream resolvers
func TestSetHostsPreservesUpstreams(t *testing.T) {
	// Create a test config file
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
hosts:
- pattern: test.local
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
		t.Fatal(err)
	}

	t.Logf("Initial upstreams: %+v", cfg.Upstreams)
	for _, u := range cfg.Upstreams {
		t.Logf("Upstream %s: type=%s, address=%s", u.Name, u.Type, u.Address)
		if u.Type == "" {
			t.Errorf("Upstream %s has empty type after initial load", u.Name)
		}
	}

	// Create proxy
	backend := &nullBackend{}
	proxy := New(cfg, backend)
	ctx := context.Background()

	// Get upstreams before SetHosts
	upstreamsBefore := proxy.GetUpstreams()
	t.Logf("Proxy upstreams before SetHosts: %v", upstreamsBefore)
	if len(upstreamsBefore) == 0 {
		t.Error("No upstreams found before SetHosts")
	}

	// Now call SetHosts to trigger the bug
	newHosts := []config.HostOverride{
		{
			Pattern: "test2.local",
			A:       []string{"127.0.0.2"},
			TTL:     60,
		},
	}

	if err := proxy.SetHosts(ctx, newHosts); err != nil {
		t.Fatal(err)
	}

	// Check upstreams in config after SetHosts
	for _, u := range cfg.Upstreams {
		t.Logf("After SetHosts - Upstream %s: type=%s, address=%s", u.Name, u.Type, u.Address)
	}

	// Get upstreams after SetHosts
	upstreamsAfter := proxy.GetUpstreams()
	t.Logf("Proxy upstreams after SetHosts: %v", upstreamsAfter)

	// This should not be empty - this is the bug!
	if len(upstreamsAfter) == 0 {
		t.Error("BUG: No upstreams found after SetHosts - upstream resolvers are broken!")
	}

	// Reload config to see what was saved
	cfg2, err := config.Load(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	t.Log("After reload from saved file:")
	for _, u := range cfg2.Upstreams {
		t.Logf("Reloaded Upstream %s: type=%s, address=%s", u.Name, u.Type, u.Address)
		if u.Type == "" {
			t.Errorf("Upstream %s has empty type after reload", u.Name)
		}
	}
}

// Test with explicit URL scheme upstreams
func TestSetHostsWithExplicitSchemes(t *testing.T) {
	// Create a test config file with explicit schemes
	testConfig := `app_name: outway
listen:
  udp: :5353
  tcp: :5353
upstreams:
- name: cloudflare-doh
  address: https://cloudflare-dns.com/dns-query
  weight: 1
- name: google-udp
  address: udp://8.8.8.8:53
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

	// Load config
	cfg, err := config.Load(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	// Create proxy and test SetHosts
	backend := &nullBackend{}
	proxy := New(cfg, backend)
	ctx := context.Background()

	upstreamsBefore := proxy.GetUpstreams()
	t.Logf("Upstreams before: %v", upstreamsBefore)
	if len(upstreamsBefore) != 2 {
		t.Errorf("Expected 2 upstreams before SetHosts, got %d", len(upstreamsBefore))
	}

	// Update hosts
	if err := proxy.SetHosts(ctx, []config.HostOverride{{Pattern: "test.local", A: []string{"127.0.0.1"}}}); err != nil {
		t.Fatal(err)
	}

	upstreamsAfter := proxy.GetUpstreams()
	t.Logf("Upstreams after: %v", upstreamsAfter)
	if len(upstreamsAfter) != 2 {
		t.Errorf("Expected 2 upstreams after SetHosts, got %d - upstreams were lost!", len(upstreamsAfter))
	}

	// Verify saved config can be reloaded
	cfg2, err := config.Load(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	for _, u := range cfg2.Upstreams {
		if u.Type == "" {
			t.Errorf("Upstream %s has empty type after reload", u.Name)
		}
	}
}

