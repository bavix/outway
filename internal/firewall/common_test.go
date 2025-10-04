package firewall_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bavix/outway/internal/firewall"
)

func TestIsSafeIfaceName(t *testing.T) {
	t.Parallel()

	tests := getSafeIfaceNameTests()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := firewall.IsSafeIfaceName(tt.iface)
			assert.Equal(t, tt.expected, result)
		})
	}
}

//nolint:funlen
func getSafeIfaceNameTests() []struct {
	name     string
	iface    string
	expected bool
} {
	return []struct {
		name     string
		iface    string
		expected bool
	}{
		{
			name:     "valid interface name",
			iface:    "eth0",
			expected: true,
		},
		{
			name:     "valid interface with underscore",
			iface:    "eth_0",
			expected: true,
		},
		{
			name:     "valid interface with dash",
			iface:    "eth-0",
			expected: true,
		},
		{
			name:     "valid interface with colon",
			iface:    "eth:0",
			expected: true,
		},
		{
			name:     "valid interface with dot",
			iface:    "eth.0",
			expected: true,
		},
		{
			name:     "valid interface with numbers",
			iface:    "eth123",
			expected: true,
		},
		{
			name:     "valid interface with mixed case",
			iface:    "Eth0",
			expected: true,
		},
		{
			name:     "empty interface name",
			iface:    "",
			expected: false,
		},
		{
			name:     "interface name with space",
			iface:    "eth 0",
			expected: false,
		},
		{
			name:     "interface name with special characters",
			iface:    "eth@0",
			expected: false,
		},
		{
			name:     "interface name with slash",
			iface:    "eth/0",
			expected: false,
		},
		{
			name:     "interface name too long",
			iface:    "verylonginterfacenamethatexceeds32chars",
			expected: false,
		},
		{
			name:     "interface name with parentheses",
			iface:    "eth(0)",
			expected: false,
		},
		{
			name:     "interface name with brackets",
			iface:    "eth[0]",
			expected: false,
		},
	}
}

func TestNormalizeIP(t *testing.T) {
	t.Parallel()

	tests := getNormalizeIPTests()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, ok := firewall.NormalizeIP(tt.ip)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.ok, ok)
		})
	}
}

//nolint:funlen
func getNormalizeIPTests() []struct {
	name     string
	ip       string
	expected string
	ok       bool
} {
	return []struct {
		name     string
		ip       string
		expected string
		ok       bool
	}{
		{
			name:     "valid IPv4",
			ip:       "192.168.1.1",
			expected: "192.168.1.1",
			ok:       true,
		},
		{
			name:     "valid IPv4 with leading zeros",
			ip:       "192.168.001.001",
			expected: "",
			ok:       false, // Go's net.ParseIP doesn't accept leading zeros
		},
		{
			name:     "valid IPv6",
			ip:       "2001:db8::1",
			expected: "2001:db8::1",
			ok:       true,
		},
		{
			name:     "valid IPv6 with brackets",
			ip:       "[2001:db8::1]",
			expected: "",
			ok:       false, // Go's net.ParseIP doesn't accept brackets
		},
		{
			name:     "invalid IP",
			ip:       "not.an.ip",
			expected: "",
			ok:       false,
		},
		{
			name:     "empty string",
			ip:       "",
			expected: "",
			ok:       false,
		},
		{
			name:     "invalid format",
			ip:       "192.168.1",
			expected: "",
			ok:       false,
		},
		{
			name:     "IPv4 mapped IPv6",
			ip:       "::ffff:192.168.1.1",
			expected: "192.168.1.1", // To4() returns the IPv4 part
			ok:       true,
		},
		{
			name:     "localhost IPv4",
			ip:       "127.0.0.1",
			expected: "127.0.0.1",
			ok:       true,
		},
		{
			name:     "localhost IPv6",
			ip:       "::1",
			expected: "::1",
			ok:       true,
		},
	}
}

func TestPFTableName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		iface    string
		expected string
	}{
		{
			name:     "simple interface",
			iface:    "eth0",
			expected: "outway_eth0",
		},
		{
			name:     "interface with underscore",
			iface:    "eth_0",
			expected: "outway_eth_0",
		},
		{
			name:     "interface with dash",
			iface:    "eth-0",
			expected: "outway_eth-0",
		},
		{
			name:     "interface with colon",
			iface:    "eth:0",
			expected: "outway_eth:0",
		},
		{
			name:     "interface with dot",
			iface:    "eth.0",
			expected: "outway_eth.0",
		},
		{
			name:     "empty interface",
			iface:    "",
			expected: "outway_",
		},
		{
			name:     "interface with numbers",
			iface:    "eth123",
			expected: "outway_eth123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := firewall.PFTableName(tt.iface)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestErrorConstants(t *testing.T) {
	t.Parallel()
	// Test error constants
	assert.Equal(t, "invalid interface name", firewall.ErrInvalidIface.Error())
	assert.Equal(t, "invalid IP address", firewall.ErrInvalidIP.Error())
}

func TestConstants(t *testing.T) {
	t.Parallel()
	// Test constants - minTTLSeconds is not exported, so we test it indirectly
	// by checking that it's used in the code
	// Placeholder test - always passes
}

func TestIfaceNameRe(t *testing.T) {
	t.Parallel()
	// Test regex pattern
	validNames := []string{
		"eth0", "eth_0", "eth-0", "eth:0", "eth.0", "eth123",
		"Eth0", "ETH0", "wlan0", "lo", "br0", "veth0",
		"en0", "en1", "en2", "en3", "en4", "en5",
		"ppp0", "tun0", "tap0", "docker0", "vboxnet0",
	}

	invalidNames := []string{
		"", "eth 0", "eth@0", "eth/0", "eth(0)", "eth[0]",
		"eth{0}", "eth|0", "eth\\0", "eth+0", "eth=0",
		"eth!0", "eth#0", "eth$0", "eth%0", "eth^0",
		"eth&0", "eth*0", "eth?0", "eth<0", "eth>0",
		"verylonginterfacenamethatexceeds32chars",
	}

	for _, name := range validNames {
		t.Run("valid_"+name, func(t *testing.T) {
			assert.True(t, firewall.IfaceNameRe.MatchString(name), "Expected %s to be valid", name)
		})
	}

	for _, name := range invalidNames {
		t.Run("invalid_"+name, func(t *testing.T) {
			t.Parallel()
			assert.False(t, firewall.IfaceNameRe.MatchString(name), "Expected %s to be invalid", name)
		})
	}
}

//nolint:funlen
func TestNormalizeIPEdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ip       string
		expected string
		ok       bool
	}{
		{
			name:     "IPv4 with port",
			ip:       "192.168.1.1:8080",
			expected: "",
			ok:       false,
		},
		{
			name:     "IPv6 with port",
			ip:       "[2001:db8::1]:8080",
			expected: "",
			ok:       false,
		},
		{
			name:     "IPv4 with CIDR",
			ip:       "192.168.1.1/24",
			expected: "",
			ok:       false,
		},
		{
			name:     "IPv6 with CIDR",
			ip:       "2001:db8::1/64",
			expected: "",
			ok:       false,
		},
		{
			name:     "zero IPv4",
			ip:       "0.0.0.0",
			expected: "0.0.0.0",
			ok:       true,
		},
		{
			name:     "zero IPv6",
			ip:       "::",
			expected: "::",
			ok:       true,
		},
		{
			name:     "private IPv4",
			ip:       "10.0.0.1",
			expected: "10.0.0.1",
			ok:       true,
		},
		{
			name:     "private IPv6",
			ip:       "fd00::1",
			expected: "fd00::1",
			ok:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, ok := firewall.NormalizeIP(tt.ip)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.ok, ok)
		})
	}
}
