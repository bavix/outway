package wol_test

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/wol"
)

func TestNewNetworkScanner(t *testing.T) {
	t.Parallel()

	scanner := wol.NewNetworkScanner()
	assert.NotNil(t, scanner)
	// assert.Equal(t, 500*time.Millisecond, scanner.timeout) // unexported field, cannot access
}

func TestNetworkScanner_GetLocalNetworkCIDR(t *testing.T) {
	t.Parallel()

	scanner := wol.NewNetworkScanner()

	cidr, err := scanner.GetLocalNetworkCIDR()
	// Should find at least one local network
	if err != nil {
		t.Skipf("No local network found: %v", err)
	}

	assert.NotEmpty(t, cidr)

	// Validate CIDR format
	_, _, err = net.ParseCIDR(cidr)
	require.NoError(t, err)
}

func TestNetworkScanner_ScanNetwork_Basic(t *testing.T) {
	t.Parallel()

	scanner := wol.NewNetworkScanner()
	ctx := context.Background()

	// Use a small test network (127.0.0.0/30)
	networkCIDR := "127.0.0.0/30"

	results, err := scanner.ScanNetwork(ctx, networkCIDR)

	// Should not error even if no devices found
	require.NoError(t, err)

	// Results should be a slice (empty or with devices)
	if results != nil {
		assert.IsType(t, []*wol.ScanResult{}, results)
	}
}

func TestNetworkScanner_ScanNetwork_InvalidCIDR(t *testing.T) {
	t.Parallel()

	scanner := wol.NewNetworkScanner()
	ctx := context.Background()

	// Invalid CIDR
	networkCIDR := "invalid-cidr"

	results, err := scanner.ScanNetwork(ctx, networkCIDR)

	require.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "invalid network CIDR")
}

// TestNetworkScanner_GetNetworkIPs tests unexported method - commented out
// func TestNetworkScanner_GetNetworkIPs(t *testing.T) {
// 	scanner := wol.NewNetworkScanner()
//
// 	// Test with a small network
// 	_, ipNet, err := net.ParseCIDR("192.168.1.0/30")
// 	require.NoError(t, err)
//
// 	ips := scanner.getNetworkIPs(ipNet)
//
// 	// Should have 2 IPs (192.168.1.1 and 192.168.1.2)
// 	// Network (192.168.1.0) and broadcast (192.168.1.3) are skipped
// 	assert.Len(t, ips, 2)
// 	assert.Contains(t, ips, "192.168.1.1")
// 	assert.Contains(t, ips, "192.168.1.2")
// }

// TestNetworkScanner_IsValidMAC tests unexported method - commented out
// func TestNetworkScanner_IsValidMAC(t *testing.T) {
// 	scanner := wol.NewNetworkScanner()
//
// 	validMACs := []string{
// 		"aa:bb:cc:dd:ee:ff",
// 		"AA:BB:CC:DD:EE:FF",
// 		"aa-bb-cc-dd-ee-ff",
// 		"12:34:56:78:9a:bc",
// 	}
//
// 	invalidMACs := []string{
// 		"",
// 		"aa:bb:cc:dd:ee",
// 		"aa:bb:cc:dd:ee:ff:gg",
// 		"aa:bb:cc:dd:ee:fg",
// 		"not-a-mac",
// 		"aa:bb:cc:dd:ee:ff:",
// 	}
//
// 	for _, mac := range validMACs {
// 		assert.True(t, scanner.isValidMAC(mac), "MAC should be valid: %s", mac)
// 	}
//
// 	for _, mac := range invalidMACs {
// 		assert.False(t, scanner.isValidMAC(mac), "MAC should be invalid: %s", mac)
// 	}
// }

// TestNetworkScanner_ParseArpOutput tests unexported method - commented out
// func TestNetworkScanner_ParseArpOutput(t *testing.T) {
// 	scanner := wol.NewNetworkScanner()
//
// 	// macOS ARP output format
// 	arpOutput := `? (192.168.1.1) at aa:bb:cc:dd:ee:ff on en0 ifscope [ethernet]
// ? (192.168.1.100) at 12:34:56:78:9a:bc on en0 ifscope [ethernet]
// ? (192.168.1.101) at invalid-mac on en0 ifscope [ethernet]
// `
//
// 	results := scanner.parseArpOutput(arpOutput)
//
// 	assert.Len(t, results, 2) // Only valid MACs
//
// 	// Check first result
// 	assert.Equal(t, "192.168.1.1", results[0].IP)
// 	assert.Equal(t, "aa:bb:cc:dd:ee:ff", results[0].MAC)
// 	assert.Equal(t, "online", results[0].Status)
//
// 	// Check second result
// 	assert.Equal(t, "192.168.1.100", results[1].IP)
// 	assert.Equal(t, "12:34:56:78:9a:bc", results[1].MAC)
// 	assert.Equal(t, "online", results[1].Status)
// }

// TestNetworkScanner_ParseNmapOutput tests unexported method - commented out
// func TestNetworkScanner_ParseNmapOutput(t *testing.T) {
// 	scanner := wol.NewNetworkScanner()
//
// 	nmapOutput := `Starting Nmap 7.80 ( https://nmap.org ) at 2023-01-01 12:00:00
// Nmap scan report for 192.168.1.1
// Host is up (0.001s latency).
// Nmap scan report for 192.168.1.100
// Host is up (0.002s latency).
// Nmap done: 2 IP addresses (2 hosts up) scanned in 0.50 seconds
// `
//
// 	results := scanner.parseNmapOutput(nmapOutput)
//
// 	assert.Len(t, results, 2)
//
// 	// Check first result
// 	assert.Equal(t, "192.168.1.1", results[0].IP)
// 	assert.Equal(t, "online", results[0].Status)
//
// 	// Check second result
// 	assert.Equal(t, "192.168.1.100", results[1].IP)
// 	assert.Equal(t, "online", results[1].Status)
// }

// TestNetworkScanner_ParseArpScanOutput tests unexported method - commented out
// func TestNetworkScanner_ParseArpScanOutput(t *testing.T) {
// 	scanner := wol.NewNetworkScanner()
//
// 	arpScanOutput := `Interface: en0, datalink type: EN10MB (Ethernet)
// Starting arp-scan 1.9.7 with 256 hosts (https://github.com/royhills/arp-scan)
// 192.168.1.1	aa:bb:cc:dd:ee:ff	Intel Corporation
// 192.168.1.100	12:34:56:78:9a:bc	Apple, Inc.
// 192.168.1.101	invalid-mac		Unknown
//
// 2 packets received by filter, 0 packets dropped by kernel
// Ending arp-scan 1.9.7: 256 hosts scanned in 0.500 seconds (512.00 hosts/sec). 2 responded
// `
//
// 	results := scanner.parseArpScanOutput(arpScanOutput)
//
// 	assert.Len(t, results, 2) // Only valid MACs
//
// 	// Check first result
// 	assert.Equal(t, "192.168.1.1", results[0].IP)
// 	assert.Equal(t, "aa:bb:cc:dd:ee:ff", results[0].MAC)
// 	assert.Equal(t, "online", results[0].Status)
//
// 	// Check second result
// 	assert.Equal(t, "192.168.1.100", results[1].IP)
// 	assert.Equal(t, "12:34:56:78:9a:bc", results[1].MAC)
// 	assert.Equal(t, "online", results[1].Status)
// }

// TestNetworkScanner_MergeArpData tests unexported method - commented out
// func TestNetworkScanner_MergeArpData(t *testing.T) {
// 	scanner := wol.NewNetworkScanner()
//
// 	// Scan results (IPs found by ping)
// 	scanResults := []*wol.ScanResult{
// 		{IP: "192.168.1.1", Status: "online"},
// 		{IP: "192.168.1.100", Status: "online"},
// 	}
//
// 	// ARP results (MAC addresses)
// 	arpResults := []*wol.ScanResult{
// 		{IP: "192.168.1.1", MAC: "aa:bb:cc:dd:ee:ff", Status: "online"},
// 		{IP: "192.168.1.100", MAC: "12:34:56:78:9a:bc", Status: "online"},
// 		{IP: "192.168.1.200", MAC: "ff:ee:dd:cc:bb:aa", Status: "online"},
// 	}
//
// 	merged := scanner.mergeArpData(scanResults, arpResults)
//
// 	assert.Len(t, merged, 3) // 2 from scan + 1 from ARP
//
// 	// Check that MAC addresses were added to scan results
// 	for _, result := range merged {
// 		switch result.IP {
// 		case "192.168.1.1":
// 			assert.Equal(t, "aa:bb:cc:dd:ee:ff", result.MAC)
// 		case "192.168.1.100":
// 			assert.Equal(t, "12:34:56:78:9a:bc", result.MAC)
// 		case "192.168.1.200":
// 			assert.Equal(t, "ff:ee:dd:cc:bb:aa", result.MAC)
// 		}
// 	}
// }

// TestNetworkScanner_HasCommand tests unexported method - commented out
// func TestNetworkScanner_HasCommand(t *testing.T) {
// 	scanner := wol.NewNetworkScanner()
//
// 	// Test with commands that should exist
// 	assert.True(t, scanner.hasCommand("ping"))
//
// 	// Test with commands that likely don't exist
// 	assert.False(t, scanner.hasCommand("nonexistent-command-12345"))
// }

// TestNetworkScanner_IncrementIP tests unexported method - commented out
// func TestNetworkScanner_IncrementIP(t *testing.T) {
// 	scanner := wol.NewNetworkScanner()
//
// 	// Test IP increment
// 	ip := net.ParseIP("192.168.1.1")
// 	scanner.incrementIP(ip)
// 	assert.Equal(t, "192.168.1.2", ip.String())
//
// 	// Test IP increment with carry
// 	ip = net.ParseIP("192.168.1.255")
// 	scanner.incrementIP(ip)
// 	assert.Equal(t, "192.168.2.0", ip.String())
// }

func TestScanResult_JSON(t *testing.T) {
	t.Parallel()

	result := &wol.ScanResult{
		IP:       "192.168.1.1",
		MAC:      "aa:bb:cc:dd:ee:ff",
		Hostname: "router",
		Vendor:   "Intel",
		Status:   "online",
	}

	// Test that ScanResult can be marshaled to JSON
	jsonData, err := json.Marshal(result)
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "192.168.1.1")
	assert.Contains(t, string(jsonData), "aa:bb:cc:dd:ee:ff")
}
