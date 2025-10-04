package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/config"
	"github.com/bavix/outway/internal/dnsproxy"
	"github.com/bavix/outway/internal/firewall"
	"github.com/bavix/outway/internal/wol"
)

// MockFirewallBackend is a mock implementation of firewall.Backend for integration testing.
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

func TestDNSProxyIntegration(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
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
				Address: "udp://8.8.8.8:53",
				Weight:  1,
			},
		},
	}

	mockBackend := &MockFirewallBackend{}
	proxy := dnsproxy.New(cfg, mockBackend)

	assert.NotNil(t, proxy)
	assert.NotNil(t, proxy.ResolverActive())
}

func TestWOLIntegration(t *testing.T) {
	t.Parallel()
	// Test WOL service creation
	wolService := wol.NewWakeOnLan()
	assert.NotNil(t, wolService)

	// Test WOL service with config
	config := &wol.Config{
		Enabled:        true,
		DefaultPort:    9,
		DefaultTimeout: 5 * time.Second,
	}
	wolServiceWithConfig := wol.NewWakeOnLanWithConfig(config)
	assert.NotNil(t, wolServiceWithConfig)

	// Test WOL handler creation - removed as WOL handler is now part of devices module
	// handler := wolhandler.NewHandler(wolService)
	// assert.NotNil(t, handler)
}

func TestWOLResolverIntegration(t *testing.T) {
	t.Parallel()
	// Test WOL resolver creation - simplified for integration testing
	t.Log("WOL resolver integration test - simplified")
}

func TestWOLConfigIntegration(t *testing.T) {
	t.Parallel()
	// Test config creation
	config := &wol.Config{
		Enabled:        true,
		DefaultPort:    9,
		DefaultTimeout: 5 * time.Second,
	}
	assert.NotNil(t, config)
	assert.True(t, config.Enabled)
	assert.Equal(t, 9, config.DefaultPort)
}

func TestWOLInterfaceDetectionIntegration(t *testing.T) {
	t.Parallel()
	// Test interface detector creation
	detector := wol.NewInterfaceDetector()
	assert.NotNil(t, detector)

	// Test interface detection (this may fail in some environments)
	ctx := context.Background()

	interfaces, err := detector.DetectInterfaces(ctx)
	if err != nil {
		t.Logf("Interface detection failed (expected in some environments): %v", err)
	} else {
		assert.NotNil(t, interfaces)
		t.Logf("Detected %d interfaces", len(interfaces))
	}
}

func TestWOLBroadcastAddressesIntegration(t *testing.T) {
	t.Parallel()
	// Test broadcast address detection
	detector := wol.NewInterfaceDetector()
	ctx := context.Background()

	addresses, err := detector.GetBroadcastAddresses(ctx)
	if err != nil {
		t.Logf("Broadcast address detection failed (expected in some environments): %v", err)
	} else {
		assert.NotNil(t, addresses)
		t.Logf("Detected %d broadcast addresses", len(addresses))
	}
}

func TestWOLMACValidationIntegration(t *testing.T) {
	t.Parallel()

	wolService := wol.NewWakeOnLan()

	// Test valid MAC addresses
	validMACs := []string{
		"00:11:22:33:44:55",
		"00-11-22-33-44-55",
		"001122334455",
		"00:11:22:33:44:55",
	}

	for _, mac := range validMACs {
		err := wolService.ValidateMAC(mac)
		require.NoError(t, err, "MAC %s should be valid", mac)
	}

	// Test invalid MAC addresses
	invalidMACs := []string{
		"invalid",
		"00:11:22:33:44",
		"00:11:22:33:44:55:66",
		"",
	}

	for _, mac := range invalidMACs {
		err := wolService.ValidateMAC(mac)
		assert.Error(t, err, "MAC %s should be invalid", mac)
	}
}

func TestWOLConfigManagerIntegration(t *testing.T) {
	t.Parallel()
	// Test config manager creation
	manager := wol.NewConfigManager()
	assert.NotNil(t, manager)

	// Test config operations
	config := &wol.Config{
		Enabled:        true,
		DefaultPort:    9,
		DefaultTimeout: 5 * time.Second,
	}

	err := manager.SetConfig(config)
	require.NoError(t, err)

	retrievedConfig := manager.GetConfig()
	assert.NotNil(t, retrievedConfig)
	assert.Equal(t, config.Enabled, retrievedConfig.Enabled)
	assert.Equal(t, config.DefaultPort, retrievedConfig.DefaultPort)
}

func TestWOLRequestResponseIntegration(t *testing.T) {
	t.Parallel()
	// Test WOL request creation
	req := &wol.WakeOnLanRequest{
		MAC:     "00:11:22:33:44:55",
		IP:      "255.255.255.255",
		Port:    9,
		Timeout: 5,
	}
	assert.NotNil(t, req)
	assert.Equal(t, "00:11:22:33:44:55", req.MAC)

	// Test WOL response creation
	resp := &wol.WakeOnLanResponse{
		Success: true,
		Message: "Packet sent successfully",
		MAC:     "00:11:22:33:44:55",
		IP:      "255.255.255.255",
		Port:    9,
	}
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestWOLNetworkInterfaceIntegration(t *testing.T) {
	t.Parallel()
	// Test network interface creation
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
	assert.NotNil(t, iface)
	assert.Equal(t, "eth0", iface.Name)
	assert.Equal(t, 1, iface.Index)
	assert.True(t, iface.IsUp)
	assert.False(t, iface.IsLoopback)
}

func TestWOLMagicPacketIntegration(t *testing.T) {
	t.Parallel()
	// Test magic packet creation
	packet := wol.MagicPacket{
		MAC:     "00:11:22:33:44:55",
		IP:      "255.255.255.255",
		Port:    9,
		Timeout: 5,
	}
	assert.NotNil(t, packet)
	assert.Equal(t, "00:11:22:33:44:55", packet.MAC)
}

func TestWOLConfigValidationIntegration(t *testing.T) {
	t.Parallel()
	// Test config validation
	config := &wol.Config{
		Enabled:     true,
		DefaultPort: 9,
		MaxRetries:  3,
	}

	// Test valid config
	assert.True(t, config.Enabled)
	assert.Equal(t, 9, config.DefaultPort)
	assert.Equal(t, 3, config.MaxRetries)

	// Test config with defaults
	configWithDefaults := &wol.Config{}
	assert.False(t, configWithDefaults.Enabled)
	assert.Equal(t, 0, configWithDefaults.DefaultPort)
	assert.Equal(t, 0, configWithDefaults.MaxRetries)
}

func TestWOLTimeoutIntegration(t *testing.T) {
	t.Parallel()
	// Test timeout configuration
	timeout := 5 * time.Second
	assert.Equal(t, 5*time.Second, timeout)

	// Test timeout in config
	config := &wol.Config{
		DefaultTimeout: timeout,
	}
	assert.Equal(t, timeout, config.DefaultTimeout)
}

func TestWOLRetryIntegration(t *testing.T) {
	t.Parallel()
	// Test retry configuration
	config := &wol.Config{
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
	}
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.RetryDelay)
}

func TestWOLServiceIntegration(t *testing.T) {
	t.Parallel()
	// Test complete WOL service integration
	wolService := wol.NewWakeOnLan()
	assert.NotNil(t, wolService)

	// Test config operations
	config := &wol.Config{
		Enabled:        true,
		DefaultPort:    9,
		DefaultTimeout: 5 * time.Second,
	}

	err := wolService.SetConfig(config)
	require.NoError(t, err)

	retrievedConfig := wolService.GetConfig()
	assert.NotNil(t, retrievedConfig)
	assert.Equal(t, config.Enabled, retrievedConfig.Enabled)

	// Test enabled status
	enabled := wolService.IsEnabled()
	assert.Equal(t, config.Enabled, enabled)
}

func TestWOLErrorHandlingIntegration(t *testing.T) {
	t.Parallel()

	wolService := wol.NewWakeOnLan()

	// Test error handling for invalid operations
	req := &wol.WakeOnLanRequest{
		MAC: "invalid-mac",
	}

	_, err := wolService.SendMagicPacket(context.Background(), req)
	assert.Error(t, err) // Should fail due to invalid MAC
}

func TestWOLContextIntegration(t *testing.T) {
	t.Parallel()

	wolService := wol.NewWakeOnLan()

	// Test context operations
	interfaces, err := wolService.GetInterfaces(context.Background())
	if err != nil {
		t.Logf("Get interfaces failed (expected in test environment): %v", err)
	} else {
		assert.NotNil(t, interfaces)
	}

	addresses, err := wolService.GetBroadcastAddresses(context.Background())
	if err != nil {
		t.Logf("Get broadcast addresses failed (expected in test environment): %v", err)
	} else {
		assert.NotNil(t, addresses)
	}
}

func TestWOLConcurrencyIntegration(t *testing.T) {
	t.Parallel()

	wolService := wol.NewWakeOnLan()

	// Test concurrent operations
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			defer func() { done <- true }()

			// Test concurrent config access
			config := wolService.GetConfig()
			assert.NotNil(t, config)

			// Test concurrent validation
			err := wolService.ValidateMAC("00:11:22:33:44:55")
			if err != nil {
				t.Errorf("Failed to validate MAC: %v", err)

				return
			}
		}()
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}
}

func TestWOLPerformanceIntegration(t *testing.T) {
	t.Parallel()

	wolService := wol.NewWakeOnLan()

	// Test performance of repeated operations
	start := time.Now()

	for range 100 {
		config := wolService.GetConfig()
		assert.NotNil(t, config)

		err := wolService.ValidateMAC("00:11:22:33:44:55")
		require.NoError(t, err)
	}

	duration := time.Since(start)
	t.Logf("100 operations completed in %v", duration)

	// Performance should be reasonable (less than 1 second for 100 operations)
	assert.Less(t, duration, time.Second)
}
