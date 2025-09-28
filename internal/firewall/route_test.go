package firewall_test

import (
	"testing"

	"github.com/bavix/outway/internal/firewall"
)

func TestRouteBackendInitialization(t *testing.T) {
	t.Parallel()

	backend, err := firewall.NewRouteBackend()
	if err != nil {
		t.Fatalf("Failed to create route backend: %v", err)
	}

	// Test basic initialization through public methods
	if backend.Name() != "route" {
		t.Errorf("Expected backend name to be 'route', got %s", backend.Name())
	}
}

func TestTunnelInfoCreation(t *testing.T) {
	t.Parallel()

	info := firewall.TunnelInfo{
		Name:     "tun0",
		TableID:  30001,
		FwMark:   30001,
		Priority: 30001,
	}

	if info.Name != "tun0" {
		t.Errorf("Expected Name to be 'tun0', got %s", info.Name)
	}

	if info.TableID != 30001 {
		t.Errorf("Expected TableID to be 30001, got %d", info.TableID)
	}

	if info.FwMark != 30001 {
		t.Errorf("Expected FwMark to be 30001, got %d", info.FwMark)
	}

	if info.Priority != 30001 {
		t.Errorf("Expected Priority to be 30001, got %d", info.Priority)
	}
}

func TestConstants(t *testing.T) {
	t.Parallel()

	if firewall.RoutingTableBase != 30000 {
		t.Errorf("Expected routingTableBase to be 30000, got %d", firewall.RoutingTableBase)
	}

	if firewall.RoutingTableMax != 30999 {
		t.Errorf("Expected routingTableMax to be 30999, got %d", firewall.RoutingTableMax)
	}

	if firewall.MarkerIPPoolStart != "169.254.0.1" {
		t.Errorf("Expected MarkerIPPoolStart to be '169.254.0.1', got %s", firewall.MarkerIPPoolStart)
	}

	if firewall.OutwayMarkerProto != "outway" {
		t.Errorf("Expected outwayMarkerProto to be 'outway', got %s", firewall.OutwayMarkerProto)
	}

	if firewall.OutwayMarkerMetric != 999 {
		t.Errorf("Expected outwayMarkerMetric to be 999, got %d", firewall.OutwayMarkerMetric)
	}
}

func TestIsSafeIfaceName(t *testing.T) {
	t.Parallel()

	validNames := []string{
		"tun0", "tun1", "eth0", "wlan0", "utun4", "utun5",
		"en0", "en1", "lo0", "bridge0", "vlan100", "wlan-0",
	}

	invalidNames := []string{
		"", "tun@0", "eth 0", "utun#4", "en;0",
		"lo0;", "bridge0|", "vlan100&", "tun0\n", "eth0\t",
	}

	for _, name := range validNames {
		if !firewall.IsSafeIfaceName(name) {
			t.Errorf("Expected %s to be valid interface name", name)
		}
	}

	for _, name := range invalidNames {
		if firewall.IsSafeIfaceName(name) {
			t.Errorf("Expected %s to be invalid interface name", name)
		}
	}
}

func TestIsIPv6(t *testing.T) {
	t.Parallel()

	ipv4Addresses := []string{
		"192.168.1.1", "10.0.0.1", "127.0.0.1", "8.8.8.8",
		"1.1.1.1", "172.16.0.1", "203.0.113.1",
	}

	ipv6Addresses := []string{
		"2001:db8::1", "::1", "fe80::1", "2001:db8:85a3::8a2e:370:7334",
		"::ffff:192.168.1.1", "[2001:db8::1]:8080", "::ffff:10.0.0.1",
	}

	for _, ip := range ipv4Addresses {
		if firewall.IsIPv6(ip) {
			t.Errorf("Expected %s to be IPv4, but IsIPv6 returned true", ip)
		}
	}

	for _, ip := range ipv6Addresses {
		if !firewall.IsIPv6(ip) {
			t.Errorf("Expected %s to be IPv6, but IsIPv6 returned false", ip)
		}
	}
}

func TestNormalizeIP(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input    string
		expected string
		valid    bool
	}{
		{"192.168.1.1", "192.168.1.1", true},
		{"[2001:db8::1]", "2001:db8::1", true},
		{"::1", "::1", true},
		{" 192.168.1.1 ", "192.168.1.1", true},
		{"fe80::1%eth0", "fe80::1", true},
		{"invalid", "", false},
		{"", "", false},
		{"192.168.1.256", "", false},
	}

	for _, tc := range testCases {
		result, valid := firewall.NormalizeIP(tc.input)
		if valid != tc.valid {
			t.Errorf("normalizeIP(%q): expected valid=%v, got %v", tc.input, tc.valid, valid)
		}

		if valid && result != tc.expected {
			t.Errorf("normalizeIP(%q): expected %q, got %q", tc.input, tc.expected, result)
		}
	}
}
