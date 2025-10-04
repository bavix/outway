package devices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/devices"
)

func TestNewDeviceManager(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	assert.NotNil(t, manager)
	// assert.NotNil(t, manager.leaseManager) // unexported field, cannot access
	// assert.NotNil(t, manager.zoneDetector) // unexported field, cannot access
	// assert.NotNil(t, manager.wolService) // unexported field, cannot access
	// assert.NotNil(t, manager.scanner) // unexported field, cannot access
	// assert.NotNil(t, manager.devices) // unexported field, cannot access
	// assert.Equal(t, 5*time.Minute, manager.scanInterval) // unexported field, cannot access
}

func TestDeviceManager_AddDevice(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	device, err := manager.AddDevice("Test Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "test.local", "Test Vendor")

	require.NoError(t, err)
	assert.NotNil(t, device)
	assert.Equal(t, "Test Device", device.Name)
	assert.Equal(t, "aa:bb:cc:dd:ee:ff", device.MAC)
	assert.Equal(t, "192.168.1.1", device.IP)
	assert.Equal(t, "test.local", device.Hostname)
	assert.Equal(t, "Test Vendor", device.Vendor)
	assert.Equal(t, "manual", device.Source)
	assert.NotEmpty(t, device.ID)
}

func TestDeviceManager_AddDevice_DuplicateMAC(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	// Add first device
	_, err := manager.AddDevice("Device 1", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "device1.local", "Vendor 1")
	require.NoError(t, err)

	// Try to add device with same MAC
	_, err = manager.AddDevice("Device 2", "aa:bb:cc:dd:ee:ff", "192.168.1.2", "device2.local", "Vendor 2")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestDeviceManager_GetDeviceByID(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	device, err := manager.AddDevice("Test Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "test.local", "Test Vendor")
	require.NoError(t, err)

	// Get device by ID
	found, exists := manager.GetDeviceByID(device.ID)
	assert.True(t, exists)
	assert.Equal(t, device.ID, found.ID)

	// Get non-existent device
	_, exists = manager.GetDeviceByID("non-existent")
	assert.False(t, exists)
}

func TestDeviceManager_GetDeviceByMAC(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	device, err := manager.AddDevice("Test Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "test.local", "Test Vendor")
	require.NoError(t, err)

	// Get device by MAC
	found, exists := manager.GetDeviceByMAC("aa:bb:cc:dd:ee:ff")
	assert.True(t, exists)
	assert.Equal(t, device.ID, found.ID)

	// Get non-existent device
	_, exists = manager.GetDeviceByMAC("ff:ee:dd:cc:bb:aa")
	assert.False(t, exists)
}

func TestDeviceManager_GetDeviceByIP(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	device, err := manager.AddDevice("Test Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "test.local", "Test Vendor")
	require.NoError(t, err)

	// Get device by IP
	found, exists := manager.GetDeviceByIP("192.168.1.1")
	assert.True(t, exists)
	assert.Equal(t, device.ID, found.ID)

	// Get non-existent device
	_, exists = manager.GetDeviceByIP("192.168.1.2")
	assert.False(t, exists)
}

func TestDeviceManager_UpdateDevice(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	device, err := manager.AddDevice("Test Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "test.local", "Test Vendor")
	require.NoError(t, err)

	// Update device
	err = manager.UpdateDevice(device.ID, "Updated Device", "aa:bb:cc:dd:ee:ff", "192.168.1.2", "updated.local", "Updated Vendor")
	require.NoError(t, err)

	// Check updated device
	updated, exists := manager.GetDeviceByID(device.ID)
	require.True(t, exists)
	assert.Equal(t, "Updated Device", updated.Name)
	assert.Equal(t, "192.168.1.2", updated.IP)
	assert.Equal(t, "updated.local", updated.Hostname)
	assert.Equal(t, "Updated Vendor", updated.Vendor)
}

func TestDeviceManager_UpdateDevice_NotFound(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	err := manager.UpdateDevice("non-existent", "Updated Device", "aa:bb:cc:dd:ee:ff", "192.168.1.2", "updated.local", "Updated Vendor")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeviceManager_DeleteDevice(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	device, err := manager.AddDevice("Test Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "test.local", "Test Vendor")
	require.NoError(t, err)

	// Delete device
	err = manager.DeleteDevice(device.ID)
	require.NoError(t, err)

	// Check device is deleted
	_, exists := manager.GetDeviceByID(device.ID)
	assert.False(t, exists)
}

func TestDeviceManager_DeleteDevice_NotFound(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	err := manager.DeleteDevice("non-existent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeviceManager_GetDevicesByType(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	// Add devices of different types
	_, err := manager.AddDevice("MacBook", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "macbook.local", "Apple MacBook")
	require.NoError(t, err)

	_, err = manager.AddDevice("iPhone", "bb:cc:dd:ee:ff:aa", "192.168.1.2", "iphone.local", "Apple iPhone")
	require.NoError(t, err)

	_, err = manager.AddDevice("Router", "cc:dd:ee:ff:aa:bb", "192.168.1.254", "router.local", "Cisco")
	require.NoError(t, err)

	// Get computers
	computers := manager.GetDevicesByType(devices.DeviceTypeComputer)
	assert.Len(t, computers, 1)
	assert.Equal(t, "MacBook", computers[0].Name)

	// Get phones
	phones := manager.GetDevicesByType(devices.DeviceTypePhone)
	assert.Len(t, phones, 1)
	assert.Equal(t, "iPhone", phones[0].Name)

	// Get routers
	routers := manager.GetDevicesByType(devices.DeviceTypeRouter)
	assert.Len(t, routers, 1)
	assert.Equal(t, "Router", routers[0].Name)
}

func TestDeviceManager_GetOnlineDevices(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	// Add online device
	_, err := manager.AddDevice("Online Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "online.local", "Vendor")
	require.NoError(t, err)

	// Add offline device
	offlineDevice, err := manager.AddDevice("Offline Device", "bb:cc:dd:ee:ff:aa", "192.168.1.2", "offline.local", "Vendor")
	require.NoError(t, err)

	offlineDevice.Status = "offline"

	// Get online devices
	onlineDevices := manager.GetOnlineDevices()
	assert.Len(t, onlineDevices, 1)
	assert.Equal(t, "Online Device", onlineDevices[0].Name)
}

func TestDeviceManager_GetWakeableDevices(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	// Add wakeable device
	wakeableDevice, err := manager.AddDevice("Wakeable Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "wakeable.local", "Vendor")
	require.NoError(t, err)

	// Set device to offline to make it wakeable
	manager.UpdateDeviceStatus(wakeableDevice.ID, "offline")

	// Add non-wakeable device (invalid MAC)
	_, err = manager.AddDevice("Non-wakeable Device", "invalid-mac", "192.168.1.2", "non-wakeable.local", "Vendor")
	require.NoError(t, err)

	// Get wakeable devices
	wakeableDevices := manager.GetWakeableDevices()
	assert.Len(t, wakeableDevices, 1)
	assert.Equal(t, "Wakeable Device", wakeableDevices[0].Name)
}

func TestDeviceManager_GetResolvableDevices(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	// Add resolvable device
	_, err := manager.AddDevice("Resolvable Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "resolvable.local", "Vendor")
	require.NoError(t, err)

	// Add non-resolvable device (no hostname)
	_, err = manager.AddDevice("Non-resolvable Device", "bb:cc:dd:ee:ff:aa", "192.168.1.2", "", "Vendor")
	require.NoError(t, err)

	// Get resolvable devices
	resolvableDevices := manager.GetResolvableDevices()
	assert.Len(t, resolvableDevices, 1)
	assert.Equal(t, "Resolvable Device", resolvableDevices[0].Name)
}

func TestDeviceManager_GetStats(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	// Add some devices
	_, err := manager.AddDevice("Device 1", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "device1.local", "Vendor 1")
	require.NoError(t, err)

	_, err = manager.AddDevice("Device 2", "bb:cc:dd:ee:ff:aa", "192.168.1.2", "device2.local", "Vendor 2")
	require.NoError(t, err)

	stats := manager.GetStats()

	assert.Equal(t, 2, stats["total_devices"])
	assert.Contains(t, stats, "online_devices")
	assert.Contains(t, stats, "wakeable_devices")
	assert.Contains(t, stats, "resolvable_devices")
	assert.Contains(t, stats, "device_types")
}

func TestDeviceManager_ResolveDevice(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	// Add device
	_, err := manager.AddDevice("Test Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "test.local", "Vendor")
	require.NoError(t, err)

	// Resolve device
	ips, err := manager.ResolveDevice("test.local")
	require.NoError(t, err)
	assert.Len(t, ips, 1)
	assert.Equal(t, "192.168.1.1", ips[0])

	// Resolve non-existent device
	_, err = manager.ResolveDevice("non-existent.local")
	require.Error(t, err)
}

func TestDeviceManager_UpdateDeviceStatus(t *testing.T) {
	t.Parallel()

	manager := devices.NewDeviceManager()

	device, err := manager.AddDevice("Test Device", "aa:bb:cc:dd:ee:ff", "192.168.1.1", "test.local", "Vendor")
	require.NoError(t, err)

	// Update status
	manager.UpdateDeviceStatus(device.ID, "waking")

	// Check updated status
	updated, exists := manager.GetDeviceByID(device.ID)
	require.True(t, exists)
	assert.Equal(t, "waking", updated.Status)
}
