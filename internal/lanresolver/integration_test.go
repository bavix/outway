package lanresolver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/miekg/dns"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/localzone"
)

// TestLANResolverIntegration tests the full flow of LAN resolver.
func TestLANResolverIntegration(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create mock DHCP leases file
	leasesFile := filepath.Join(tmpDir, "dhcp.leases")
	leasesContent := `1633024800 aa:bb:cc:dd:ee:ff 192.168.1.100 myhost *
1633024900 11:22:33:44:55:66 192.168.1.101 router id-123
1633025000 aa:aa:bb:bb:cc:cc 2001:db8::1 ipv6host *
`
	if err := os.WriteFile(leasesFile, []byte(leasesContent), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create mock UCI config file
	uciFile := filepath.Join(tmpDir, "dhcp")
	uciContent := `config dnsmasq
	option domain 'lan'
	option local '/home/'
`
	if err := os.WriteFile(uciFile, []byte(uciContent), 0o600); err != nil {
		t.Fatal(err)
	}

	// Detect zones
	zones := localzone.DetectLocalZones(nil, uciFile, "")
	if len(zones) < 1 {
		t.Fatalf("expected at least 1 zone, got %d", len(zones))
	}
	t.Logf("Detected zones: %v", zones)

	// Parse leases
	leases, err := ParseLeases(leasesFile)
	if err != nil {
		t.Fatal(err)
	}
	if len(leases) != 3 {
		t.Fatalf("expected 3 leases, got %d", len(leases))
	}

	// Create mock next resolver (always returns error to ensure LAN resolver handles it)
	mockNext := &mockResolver{shouldFail: true}

	// Create LAN resolver
	cfg := &config.Config{}
	resolver := NewResolver(mockNext, zones, cfg)
	resolver.UpdateLeases(leases, zones)

	ctx := context.Background()

	// Test 1: Resolve myhost.lan (should find it in leases)
	t.Run("resolve_myhost_lan", func(t *testing.T) {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn("myhost.lan"), dns.TypeA)

		resp, src, err := resolver.Resolve(ctx, m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if src != "lan-resolver" {
			t.Errorf("expected source 'lan-resolver', got '%s'", src)
		}
		if resp.Rcode != dns.RcodeSuccess {
			t.Errorf("expected NOERROR, got %s", dns.RcodeToString[resp.Rcode])
		}
		if len(resp.Answer) != 1 {
			t.Fatalf("expected 1 answer, got %d", len(resp.Answer))
		}

		// Check answer is correct IP
		if a, ok := resp.Answer[0].(*dns.A); ok {
			if a.A.String() != "192.168.1.100" {
				t.Errorf("expected IP 192.168.1.100, got %s", a.A.String())
			}
		} else {
			t.Error("expected A record")
		}
	})

	// Test 2: Resolve router.home (should find it)
	t.Run("resolve_router_home", func(t *testing.T) {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn("router.home"), dns.TypeA)

		resp, src, err := resolver.Resolve(ctx, m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if src != "lan-resolver" {
			t.Errorf("expected source 'lan-resolver', got '%s'", src)
		}
		if resp.Rcode != dns.RcodeSuccess {
			t.Errorf("expected NOERROR, got %s", dns.RcodeToString[resp.Rcode])
		}
		if len(resp.Answer) != 1 {
			t.Fatalf("expected 1 answer, got %d", len(resp.Answer))
		}
	})

	// Test 3: Resolve ipv6host.lan for AAAA (should find IPv6)
	t.Run("resolve_ipv6host_aaaa", func(t *testing.T) {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn("ipv6host.lan"), dns.TypeAAAA)

		resp, src, err := resolver.Resolve(ctx, m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if src != "lan-resolver" {
			t.Errorf("expected source 'lan-resolver', got '%s'", src)
		}
		if resp.Rcode != dns.RcodeSuccess {
			t.Errorf("expected NOERROR, got %s", dns.RcodeToString[resp.Rcode])
		}
		if len(resp.Answer) != 1 {
			t.Fatalf("expected 1 answer, got %d", len(resp.Answer))
		}

		// Check answer is correct IPv6
		if aaaa, ok := resp.Answer[0].(*dns.AAAA); ok {
			if aaaa.AAAA.String() != "2001:db8::1" {
				t.Errorf("expected IP 2001:db8::1, got %s", aaaa.AAAA.String())
			}
		} else {
			t.Error("expected AAAA record")
		}
	})

	// Test 4: Resolve nonexistent.lan (should return NXDOMAIN)
	t.Run("resolve_nonexistent_nxdomain", func(t *testing.T) {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn("nonexistent.lan"), dns.TypeA)

		resp, src, err := resolver.Resolve(ctx, m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if src != "lan-resolver" {
			t.Errorf("expected source 'lan-resolver', got '%s'", src)
		}
		if resp.Rcode != dns.RcodeNameError {
			t.Errorf("expected NXDOMAIN, got %s", dns.RcodeToString[resp.Rcode])
		}
		if len(resp.Answer) != 0 {
			t.Errorf("expected 0 answers, got %d", len(resp.Answer))
		}
	})

	// Test 5: Resolve example.com (not local zone, should pass through)
	t.Run("resolve_external_passthrough", func(t *testing.T) {
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn("example.com"), dns.TypeA)

		_, src, err := resolver.Resolve(ctx, m)
		// Mock resolver is set to fail, so we expect error
		if err == nil {
			t.Error("expected error from mock resolver")
		}
		// Source should be from mock, not lan-resolver
		if src == "lan-resolver" {
			t.Error("should not have been handled by lan-resolver")
		}
	})

	// Test 6: Verify no upstream forwarding for local zones
	t.Run("no_upstream_forwarding", func(t *testing.T) {
		// Reset mock call counter
		mockNext.callCount = 0

		// Query local zone
		m := new(dns.Msg)
		m.SetQuestion(dns.Fqdn("myhost.lan"), dns.TypeA)
		_, _, _ = resolver.Resolve(ctx, m)

		// Mock should not have been called
		if mockNext.callCount > 0 {
			t.Error("local zone query was forwarded to upstream")
		}
	})
}

// mockResolver is a test resolver that tracks calls.
type mockResolver struct {
	shouldFail bool
	callCount  int
}

func (m *mockResolver) Resolve(_ context.Context, q *dns.Msg) (*dns.Msg, string, error) {
	m.callCount++
	if m.shouldFail {
		resp := new(dns.Msg)
		resp.SetRcode(q, dns.RcodeServerFailure)
		return resp, "mock", context.DeadlineExceeded
	}

	resp := new(dns.Msg)
	resp.SetReply(q)
	return resp, "mock", nil
}
