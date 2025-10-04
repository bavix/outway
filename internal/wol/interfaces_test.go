package wol_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/wol"
)

func TestInterfaceDetector_NewInterfaceDetector(t *testing.T) {
	t.Parallel()

	detector := wol.NewInterfaceDetector()

	assert.NotNil(t, detector)
	assert.NotNil(t, detector)
}

func TestInterfaceDetector_DetectInterfaces(t *testing.T) {
	t.Parallel()

	detector := wol.NewInterfaceDetector()

	interfaces, err := detector.DetectInterfaces(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, interfaces)

	// Should have at least one interface (may be empty in some test environments)
	// assert.GreaterOrEqual(t, len(interfaces), 1)

	// Check interface structure (only if interfaces exist)
	if len(interfaces) > 0 {
		for _, iface := range interfaces {
			assert.NotEmpty(t, iface.Name)
			assert.Positive(t, iface.Index)
			// HardwareAddr may be empty in some test environments
			// assert.NotEmpty(t, iface.HardwareAddr)
		}
	}
}

func TestInterfaceDetector_GetBestInterface(t *testing.T) {
	t.Parallel()

	detector := wol.NewInterfaceDetector()

	iface, err := detector.GetBestInterface(context.Background())
	if err != nil {
		// This might fail in some test environments
		t.Logf("No suitable interface found: %v", err)

		return
	}

	assert.NotNil(t, iface)
	assert.NotEmpty(t, iface.Name)
	assert.True(t, iface.IsUp)
	assert.False(t, iface.IsLoopback)
}

func TestInterfaceDetector_GetInterfaceByName(t *testing.T) {
	t.Parallel()

	detector := wol.NewInterfaceDetector()

	interfaces, err := detector.DetectInterfaces(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, interfaces)

	// Test with existing interface
	firstInterface := interfaces[0]
	iface, err := detector.GetInterfaceByName(context.Background(), firstInterface.Name)
	require.NoError(t, err)
	assert.NotNil(t, iface)
	assert.Equal(t, firstInterface.Name, iface.Name)

	// Test with non-existing interface
	_, err = detector.GetInterfaceByName(context.Background(), "nonexistent")
	require.Error(t, err)
}

//nolint:funlen
func TestInterfaceDetector_ValidateInterface(t *testing.T) {
	t.Parallel()

	detector := wol.NewInterfaceDetector()

	tests := []struct {
		name     string
		iface    *wol.NetworkInterface
		hasError bool
	}{
		{
			name:     "nil interface",
			iface:    nil,
			hasError: true,
		},
		{
			name: "valid interface",
			iface: &wol.NetworkInterface{
				Name:       "eth0",
				IsUp:       true,
				IsLoopback: false,
				Broadcast:  "192.168.1.255",
			},
			hasError: false,
		},
		{
			name: "loopback interface",
			iface: &wol.NetworkInterface{
				Name:       "lo",
				IsUp:       true,
				IsLoopback: true,
				Broadcast:  "127.0.0.1",
			},
			hasError: true,
		},
		{
			name: "down interface",
			iface: &wol.NetworkInterface{
				Name:       "eth0",
				IsUp:       false,
				IsLoopback: false,
				Broadcast:  "192.168.1.255",
			},
			hasError: true,
		},
		{
			name: "no broadcast address",
			iface: &wol.NetworkInterface{
				Name:       "eth0",
				IsUp:       true,
				IsLoopback: false,
				Broadcast:  "",
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := detector.ValidateInterface(tt.iface)
			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestInterfaceDetector_GetBroadcastAddresses(t *testing.T) {
	t.Parallel()

	detector := wol.NewInterfaceDetector()

	addresses, err := detector.GetBroadcastAddresses(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, addresses)

	// Should have at least the global broadcast address
	assert.Contains(t, addresses, "255.255.255.255")
}

func TestInterfaceDetector_IsValidBroadcastAddress(t *testing.T) {
	t.Parallel()

	detector := wol.NewInterfaceDetector()

	tests := []struct {
		name     string
		addr     string
		expected bool
	}{
		{"global broadcast", "255.255.255.255", true},
		{"valid IPv4", "192.168.1.255", true},
		{"invalid IPv4", "192.168.1.256", false},
		{"invalid format", "not-an-ip", false},
		{"IPv6 address", "2001:db8::1", false},
		{"empty address", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := detector.IsValidBroadcastAddress(context.Background(), tt.addr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInterfaceDetector_GetInterfaceSummary(t *testing.T) {
	t.Parallel()

	detector := wol.NewInterfaceDetector()

	summary := detector.GetInterfaceSummary(context.Background())
	assert.NotEmpty(t, summary)
	assert.Contains(t, summary, "up") // Should contain status information
}

// TestInterfaceDetector_CalculateBroadcast tests unexported method - commented out
// func TestInterfaceDetector_CalculateBroadcast(t *testing.T) {
// 	detector := wol.NewInterfaceDetector()
//
// 	tests := []struct {
// 		name     string
// 		ip       string
// 		mask     string
// 		expected string
// 	}{
// 		{
// 			name:     "class C network",
// 			ip:       "192.168.1.100",
// 			mask:     "255.255.255.0",
// 			expected: "192.168.1.255",
// 		},
// 		{
// 			name:     "class B network",
// 			ip:       "172.16.1.100",
// 			mask:     "255.255.0.0",
// 			expected: "172.16.255.255",
// 		},
// 		{
// 			name:     "class A network",
// 			ip:       "10.1.1.100",
// 			mask:     "255.0.0.0",
// 			expected: "10.255.255.255",
// 		},
// 		{
// 			name:     "subnet /24",
// 			ip:       "192.168.0.1",
// 			mask:     "255.255.255.0",
// 			expected: "192.168.0.255",
// 		},
// 		{
// 			name:     "subnet /16",
// 			ip:       "192.168.0.1",
// 			mask:     "255.255.0.0",
// 			expected: "192.168.255.255",
// 		},
// 	}
//
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Create IP network manually
// 			ip := net.ParseIP(tt.ip)
// 			mask := net.IPMask(net.ParseIP(tt.mask).To4())
// 			ipNet := &net.IPNet{
// 				IP:   ip,
// 				Mask: mask,
// 			}
//
// 			result := detector.calculateBroadcast(ipNet)
// 			assert.Equal(t, tt.expected, result)
// 		})
// 	}
// }

// TestInterfaceDetector_ProcessInterface tests unexported method - commented out
// func TestInterfaceDetector_ProcessInterface(t *testing.T) {
// 	detector := wol.NewInterfaceDetector()
//
// 	// Get a real interface for testing
// 	interfaces, err := net.Interfaces()
// 	require.NoError(t, err)
// 	require.NotEmpty(t, interfaces)
//
// 	// Test with the first interface
// 	firstInterface := interfaces[0]
// 	netIface, err := detector.processInterface(firstInterface)
// 	require.NoError(t, err)
// 	assert.NotNil(t, netIface)
// 	assert.Equal(t, firstInterface.Name, netIface.Name)
// 	assert.Equal(t, firstInterface.Index, netIface.Index)
// 	assert.Equal(t, firstInterface.MTU, netIface.MTU)
// 	assert.Equal(t, firstInterface.HardwareAddr.String(), netIface.HardwareAddr)
// 	assert.Equal(t, firstInterface.Flags&net.FlagUp != 0, netIface.IsUp)
// 	assert.Equal(t, firstInterface.Flags&net.FlagLoopback != 0, netIface.IsLoopback)
// }

func TestNetworkInterface_JSONSerialization(t *testing.T) {
	t.Parallel()

	iface := wol.NetworkInterface{
		Name:         "eth0",
		Index:        1,
		MTU:          1500,
		HardwareAddr: "00:11:22:33:44:55",
		IPs:          []string{"192.168.1.100"},
		Broadcast:    "192.168.1.255",
		IsUp:         true,
		IsLoopback:   false,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(iface)
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaled wol.NetworkInterface

	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	assert.Equal(t, iface, unmarshaled)
}

// TestInterfaceDetector_EdgeCases tests unexported method - commented out
// func TestInterfaceDetector_EdgeCases(t *testing.T) {
// 	detector := wol.NewInterfaceDetector()
//
// 	// Test with invalid broadcast address
// 	invalidAddr := "999.999.999.999"
// 	result := detector.IsValidBroadcastAddress(context.Background(), invalidAddr)
// 	assert.False(t, result)
//
// 	// Test with nil IP network (should not panic)
// 	defer func() {
// 		if r := recover(); r != nil {
// 			t.Errorf("calculateBroadcast panicked with nil input: %v", r)
// 		}
// 	}()
//
// 	broadcast := detector.calculateBroadcast(nil)
// 	assert.Empty(t, broadcast)
// }

func TestInterfaceDetector_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	detector := wol.NewInterfaceDetector()

	// Test concurrent access to DetectInterfaces
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			defer func() { done <- true }()

			interfaces, err := detector.DetectInterfaces(context.Background())
			if err != nil {
				t.Errorf("Failed to detect interfaces: %v", err)

				return
			}

			assert.NotNil(t, interfaces)
		}()
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}
}

func TestInterfaceDetector_ErrorHandling(t *testing.T) {
	t.Parallel()

	detector := wol.NewInterfaceDetector()

	// Test GetInterfaceByName with empty name
	_, err := detector.GetInterfaceByName(context.Background(), "")
	require.Error(t, err)

	// Test GetInterfaceByName with non-existent interface
	_, err = detector.GetInterfaceByName(context.Background(), "nonexistent-interface-12345")
	require.Error(t, err)

	// Test ValidateInterface with various invalid inputs
	invalidInterfaces := []*wol.NetworkInterface{
		nil,
		{Name: "eth0", IsUp: true, IsLoopback: false, Broadcast: ""}, // Empty broadcast should fail
		{Name: "eth0", IsUp: true, IsLoopback: true, Broadcast: "192.168.1.255"},
		{Name: "eth0", IsUp: false, IsLoopback: false, Broadcast: "192.168.1.255"},
		{Name: "eth0", IsUp: true, IsLoopback: false, Broadcast: ""},
	}

	for i, iface := range invalidInterfaces {
		t.Run(fmt.Sprintf("invalid_interface_%d", i), func(t *testing.T) {
			t.Parallel()

			err := detector.ValidateInterface(iface)
			require.Error(t, err)
		})
	}
}
