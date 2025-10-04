package devices

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"

	customerrors "github.com/bavix/outway/internal/errors"
	"github.com/bavix/outway/internal/lanresolver"
	"github.com/bavix/outway/internal/localzone"
	"github.com/bavix/outway/internal/wol"
)

const (
	defaultScanInterval = 5 * time.Minute
	defaultTimeout      = 10 * time.Second
	wakeTimeout         = 30 * time.Second
)

// DeviceType represents the type of a network device.
type DeviceType string

// Device represents a network device with DNS and WOL capabilities.
type Device struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	MAC       string     `json:"mac"`
	IP        string     `json:"ip"`
	Hostname  string     `json:"hostname"`
	Vendor    string     `json:"vendor"`
	Type      DeviceType `json:"type"`
	Status    string     `json:"status"` // StatusOnline, StatusOffline, StatusUnknown
	LastSeen  time.Time  `json:"last_seen"`
	Source    string     `json:"source"`           // dhcp, scan, manual
	Expire    *time.Time `json:"expire,omitempty"` // lease expiration
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// Clone creates a deep copy of the device.
func (d *Device) Clone() *Device {
	return &Device{
		ID:        d.ID,
		Name:      d.Name,
		MAC:       d.MAC,
		IP:        d.IP,
		Hostname:  d.Hostname,
		Vendor:    d.Vendor,
		Type:      d.Type,
		Status:    d.Status,
		LastSeen:  d.LastSeen,
		Source:    d.Source,
		Expire:    d.Expire,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// FileWatcher is a placeholder for file watching functionality.
type FileWatcher struct {
	// This is a placeholder - actual implementation would be added later
}

// Close closes the file watcher.
func (fw *FileWatcher) Close() error {
	return nil
}

// GetDisplayInfo returns device information formatted for display.
func (d *Device) GetDisplayInfo() map[string]interface{} {
	if d == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"id":        d.ID,
		"name":      d.Name,
		"mac":       d.MAC,
		"ip":        d.IP,
		"hostname":  d.Hostname,
		"vendor":    d.Vendor,
		"type":      string(d.Type),
		"status":    d.Status,
		"last_seen": d.LastSeen,
	}
}

// GetDeviceType returns the device type.
func (d *Device) GetDeviceType() DeviceType {
	return d.Type
}

// IsOnline checks if the device is online.
func (d *Device) IsOnline() bool {
	return d.Status == "online"
}

// CanBeWoken checks if the device can be woken up.
// We can only wake devices that have a valid MAC address.
func (d *Device) CanBeWoken() bool {
	return d.MAC != "" && d.IsValidMAC()
}

// CanBeResolved checks if the device can be resolved via DNS.
func (d *Device) CanBeResolved() bool {
	return d.Hostname != "" && d.IP != ""
}

// GetDisplayName returns a display name for the device.
func (d *Device) GetDisplayName() string {
	if d.Name != "" {
		return d.Name
	}

	if d.Hostname != "" {
		return d.Hostname
	}

	return d.MAC
}

// CanWake checks if the device can be woken up.
func (d *Device) CanWake() bool {
	return d.CanBeWoken()
}

// IsValidMAC checks if the MAC address is valid.
func (d *Device) IsValidMAC() bool {
	return len(d.MAC) == 17 && strings.Count(d.MAC, ":") == 5
}

// CanResolve checks if the device can be resolved via DNS.
func (d *Device) CanResolve() bool {
	return d.CanBeResolved()
}

// IsLocal checks if the device is on the local network.
func (d *Device) IsLocal() bool {
	return strings.HasPrefix(d.IP, "192.168.") ||
		strings.HasPrefix(d.IP, "10.") ||
		strings.HasPrefix(d.IP, "172.")
}

// GetDeviceType determines the device type based on vendor and hostname.
func GetDeviceType(vendor, hostname string) DeviceType {
	vendorLower := strings.ToLower(vendor)
	hostnameLower := strings.ToLower(hostname)

	// Check for specific device types first
	if isAppleDevice(hostnameLower) {
		return DeviceTypePhone
	}

	if isAppleComputer(vendorLower) {
		return DeviceTypeComputer
	}

	if isPhoneDevice(vendorLower) {
		return DeviceTypePhone
	}

	if isRouterDevice(vendorLower) {
		return DeviceTypeRouter
	}

	if isTVDevice(vendorLower) {
		return DeviceTypeTV
	}

	return DeviceTypeOther
}

func isAppleDevice(hostnameLower string) bool {
	return strings.Contains(hostnameLower, "iphone") || strings.Contains(hostnameLower, "ipad")
}

func isAppleComputer(vendorLower string) bool {
	return strings.Contains(vendorLower, "apple")
}

func isPhoneDevice(vendorLower string) bool {
	return strings.Contains(vendorLower, "samsung") || strings.Contains(vendorLower, "huawei") ||
		strings.Contains(vendorLower, "xiaomi") || strings.Contains(vendorLower, "oneplus")
}

func isRouterDevice(vendorLower string) bool {
	return strings.Contains(vendorLower, "cisco") || strings.Contains(vendorLower, "netgear") ||
		strings.Contains(vendorLower, "tp-link") || strings.Contains(vendorLower, "asus")
}

func isTVDevice(vendorLower string) bool {
	return strings.Contains(vendorLower, "lg") || strings.Contains(vendorLower, "samsung")
}

// DeviceManager manages all network devices with DNS and WOL capabilities.
type DeviceManager struct {
	// Core components
	leaseManager *lanresolver.LeaseManager
	zoneDetector *localzone.ZoneDetector
	wolService   *wol.WakeOnLan
	scanner      *wol.NetworkScanner
	fileWatcher  *FileWatcher

	// Storage
	devices map[string]*Device
	mu      sync.RWMutex

	// Auto-scan configuration
	scanInterval time.Duration
	lastScan     time.Time
}

// GetCommonLeasePaths returns common DHCP lease file paths.
func GetCommonLeasePaths() []string {
	return []string{
		"/tmp/dhcp.leases",                    // OpenWrt
		"/var/lib/dhcp/dhcpd.leases",          // ISC DHCP
		"/var/lib/dhcp/dhcp.leases",           // ISC DHCP alternative
		"/var/lib/dhcpcd/dhcpcd.leases",       // dhcpcd
		"/var/lib/NetworkManager/dhcp.leases", // NetworkManager
	}
}

// detectDHCPLeasePath automatically detects the best DHCP lease file path.
func detectDHCPLeasePath() string {
	paths := GetCommonLeasePaths()

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Default to OpenWrt path if no file found
	return "/tmp/dhcp.leases"
}

// NewDeviceManager creates a new device manager with automatic configuration.
func NewDeviceManager() *DeviceManager {
	// Create components with automatic configuration
	zoneDetector := localzone.NewZoneDetector()
	leasePath := detectDHCPLeasePath()
	leaseManager := lanresolver.NewLeaseManager(leasePath)
	wolService := wol.NewWakeOnLan()
	scanner := wol.NewNetworkScanner()

	dm := &DeviceManager{
		leaseManager: leaseManager,
		zoneDetector: zoneDetector,
		wolService:   wolService,
		scanner:      scanner,
		devices:      make(map[string]*Device),
		mu:           sync.RWMutex{},
		scanInterval: defaultScanInterval, // Auto-scan every 5 minutes
	}

	// Debug: check if dm is properly initialized
	if dm.devices == nil {
		panic("dm.devices is nil")
	}

	// Note: Initial DHCP lease loading is deferred to first use
	// This avoids creating context.Background() in constructor

	// File watcher is disabled by default to avoid resource leaks
	// It can be enabled manually if needed
	// Common DHCP lease file paths:
	// - /var/lib/dhcp/dhcpd.leases (ISC DHCP)
	// - /var/lib/dhcp/dhcp.leases (ISC DHCP alternative)
	// - /tmp/dhcp.leases (OpenWrt)
	// - /var/lib/dhcpcd/dhcpcd.leases (dhcpcd)

	// Start auto-scan in background (disabled for tests)
	// go dm.autoScanLoop()

	return dm
}

// GetAllDevices returns all managed devices.
// Note: This method doesn't load DHCP leases automatically.
// Use ScanNetwork() or RefreshDHCPLeases() to load data first.
func (dm *DeviceManager) GetAllDevices() []*Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	devices := make([]*Device, 0, len(dm.devices))
	for _, device := range dm.devices {
		devices = append(devices, device)
	}

	return devices
}

// ScanNetwork performs a network scan and updates the device list.
func (dm *DeviceManager) ScanNetwork(ctx context.Context) ([]*Device, error) {
	if dm.scanner == nil {
		return nil, customerrors.ErrNetworkScannerNotInitialized
	}

	// Update DHCP leases first
	dm.loadFromDHCPLeases(ctx)

	// Get local network CIDR
	networkCIDR, err := dm.scanner.GetLocalNetworkCIDR()
	if err != nil {
		return nil, fmt.Errorf("failed to get local network: %w", err)
	}

	// Perform scan
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	scanResults, err := dm.scanner.ScanNetwork(ctx, networkCIDR)
	if err != nil {
		return nil, fmt.Errorf("network scan failed: %w", err)
	}

	// Convert scan results to devices and update storage
	dm.mu.Lock()
	defer dm.mu.Unlock()

	devices := make([]*Device, 0, len(dm.devices))

	for _, result := range scanResults {
		// Generate a better name if hostname is empty
		name := result.Hostname
		if name == "" {
			name = "Device-" + result.IP
		}

		device := &Device{
			ID:        fmt.Sprintf("%s-%s", result.IP, result.MAC),
			Name:      name,
			MAC:       result.MAC,
			IP:        result.IP,
			Hostname:  result.Hostname,
			Vendor:    result.Vendor,
			Type:      GetDeviceType(result.Vendor, result.Hostname),
			Status:    result.Status,
			LastSeen:  time.Now(),
			Source:    "scan",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Update or add device
		dm.devices[device.ID] = device
		devices = append(devices, device)
	}

	return devices, nil
}

// GetDeviceByID returns a device by ID.
func (dm *DeviceManager) GetDeviceByID(id string) (*Device, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	device, exists := dm.devices[id]

	return device, exists
}

// GetDeviceByMAC returns a device by MAC address.
func (dm *DeviceManager) GetDeviceByMAC(mac string) (*Device, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for _, device := range dm.devices {
		if device.MAC == mac {
			return device, true
		}
	}

	return nil, false
}

// GetDeviceByIP returns a device by IP address.
func (dm *DeviceManager) GetDeviceByIP(ip string) (*Device, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	for _, device := range dm.devices {
		if device.IP == ip {
			return device, true
		}
	}

	return nil, false
}

// GetDevicesByType returns devices filtered by type.
func (dm *DeviceManager) GetDevicesByType(deviceType DeviceType) []*Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	devices := make([]*Device, 0, len(dm.devices))
	for _, device := range dm.devices {
		if device.GetDeviceType() == deviceType {
			devices = append(devices, device)
		}
	}

	return devices
}

// GetOnlineDevices returns only online devices.
func (dm *DeviceManager) GetOnlineDevices() []*Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	devices := make([]*Device, 0, len(dm.devices))
	for _, device := range dm.devices {
		if device.IsOnline() {
			devices = append(devices, device)
		}
	}

	return devices
}

// GetWakeableDevices returns devices that can be woken up.
func (dm *DeviceManager) GetWakeableDevices() []*Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	devices := make([]*Device, 0, len(dm.devices))
	for _, device := range dm.devices {
		if device.CanBeWoken() {
			devices = append(devices, device)
		}
	}

	return devices
}

// GetResolvableDevices returns devices that can be resolved via DNS.
func (dm *DeviceManager) GetResolvableDevices() []*Device {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	devices := make([]*Device, 0, len(dm.devices))
	for _, device := range dm.devices {
		if device.CanBeResolved() {
			devices = append(devices, device)
		}
	}

	return devices
}

// ScanDevices performs a network scan and updates device information.
func (dm *DeviceManager) ScanDevices(ctx context.Context) ([]*Device, error) {
	logger := zerolog.Ctx(ctx)

	// Get local network
	networkCIDR, err := dm.scanner.GetLocalNetworkCIDR()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get local network, using DHCP only")

		return dm.loadFromDHCPLeases(ctx), nil
	}

	logger.Info().
		Str("network", networkCIDR).
		Msg("Starting device scan")

		// Perform network scan
	scanCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	scanResults, err := dm.scanner.ScanNetwork(scanCtx, networkCIDR)
	if err != nil {
		logger.Warn().Err(err).Msg("Network scan failed, using DHCP only")

		return dm.loadFromDHCPLeases(ctx), nil
	}

	// Update devices from scan results
	updatedDevices := dm.updateFromScanResults(scanResults)

	// Also load from DHCP leases for devices not found in scan
	dhcpDevices := dm.loadFromDHCPLeases(ctx)

	// Merge results
	allDevices := make([]*Device, 0, len(updatedDevices)+len(dhcpDevices))
	allDevices = append(allDevices, updatedDevices...)
	allDevices = append(allDevices, dhcpDevices...)

	logger.Info().
		Int("total_devices", len(dm.devices)).
		Int("scan_devices", len(updatedDevices)).
		Int("dhcp_devices", len(dhcpDevices)).
		Msg("Device scan completed")

	dm.lastScan = time.Now()

	return allDevices, nil
}

// WakeDevice wakes up a device by ID.
func (dm *DeviceManager) WakeDevice(ctx context.Context, deviceID string) error {
	device, exists := dm.GetDeviceByID(deviceID)
	if !exists {
		return customerrors.ErrDeviceNotFoundWithID(deviceID)
	}

	if !device.CanBeWoken() {
		return customerrors.ErrDeviceCannotBeWokenUp(device.GetDisplayName())
	}

	// Create WOL request
	req := &wol.WakeOnLanRequest{
		MAC: device.MAC,
	}

	// Send magic packet
	_, err := dm.wolService.SendMagicPacket(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to wake device %s: %w", device.GetDisplayName(), err)
	}

	// Update device status
	dm.UpdateDeviceStatus(deviceID, "waking")

	return nil
}

// WakeAllDevices wakes up all wakeable devices.
func (dm *DeviceManager) WakeAllDevices(ctx context.Context) ([]*Device, error) {
	wakeableDevices := dm.GetWakeableDevices()

	wokenDevices := make([]*Device, 0, len(wakeableDevices))

	for _, device := range wakeableDevices {
		if err := dm.WakeDevice(ctx, device.ID); err != nil {
			zerolog.Ctx(ctx).Warn().
				Str("device", device.GetDisplayName()).
				Err(err).
				Msg("Failed to wake device")

			continue
		}

		wokenDevices = append(wokenDevices, device)
	}

	return wokenDevices, nil
}

// ResolveDevice resolves a device hostname to IP addresses.
func (dm *DeviceManager) ResolveDevice(hostname string) ([]string, error) {
	// Try to find device by hostname
	for _, device := range dm.GetAllDevices() {
		if device.Hostname == hostname {
			return []string{device.IP}, nil
		}
	}

	// Fallback to DHCP lease resolution
	lease := dm.leaseManager.GetLease(hostname)
	if lease != nil {
		return []string{lease.IP}, nil
	}

	return nil, customerrors.ErrDeviceNotFoundWithHostname(hostname)
}

// AddDevice manually adds a new device.
func (dm *DeviceManager) AddDevice(name, mac, ip, hostname, vendor string) (*Device, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Check if device already exists (inline check to avoid deadlock)
	for _, device := range dm.devices {
		if device.MAC == mac {
			return nil, customerrors.ErrDeviceAlreadyExistsWithMAC(mac)
		}
	}

	device := &Device{
		ID:        dm.generateDeviceID(),
		Name:      name,
		MAC:       mac,
		IP:        ip,
		Hostname:  hostname,
		Vendor:    vendor,
		Type:      GetDeviceType(vendor, hostname),
		Status:    "online", // Default to online for manually added devices
		Source:    "manual",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	// Determine capabilities
	dm.updateDeviceCapabilities(device)

	dm.devices[device.ID] = device

	return device, nil
}

// UpdateDevice updates an existing device.
func (dm *DeviceManager) UpdateDevice(id, name, mac, ip, hostname, vendor string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	device, exists := dm.devices[id]
	if !exists {
		return customerrors.ErrDeviceNotFoundWithID(id)
	}

	device.Name = name
	device.MAC = mac
	device.IP = ip
	device.Hostname = hostname
	device.Vendor = vendor
	device.UpdatedAt = time.Now()

	// Update capabilities
	dm.updateDeviceCapabilities(device)

	return nil
}

// DeleteDevice removes a device.
func (dm *DeviceManager) DeleteDevice(id string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.devices[id]; !exists {
		return customerrors.ErrDeviceNotFoundWithID(id)
	}

	delete(dm.devices, id)

	return nil
}

// GetStats returns device statistics.
func (dm *DeviceManager) GetStats() map[string]interface{} {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_devices":      len(dm.devices),
		"online_devices":     0,
		"wakeable_devices":   0,
		"resolvable_devices": 0,
		"last_scan":          dm.lastScan,
		"device_types":       make(map[DeviceType]int),
	}

	for _, device := range dm.devices {
		if device.IsOnline() {
			if onlineCount, ok := stats["online_devices"].(int); ok {
				stats["online_devices"] = onlineCount + 1
			}
		}

		if device.CanWake() {
			if wakeableCount, ok := stats["wakeable_devices"].(int); ok {
				stats["wakeable_devices"] = wakeableCount + 1
			}
		}

		if device.CanResolve() {
			if resolvableCount, ok := stats["resolvable_devices"].(int); ok {
				stats["resolvable_devices"] = resolvableCount + 1
			}
		}

		deviceType := device.GetDeviceType()
		if deviceTypes, ok := stats["device_types"].(map[DeviceType]int); ok {
			deviceTypes[deviceType]++
		}
	}

	return stats
}

// Private methods

// UpdateDeviceStatus updates the status of a device.
func (dm *DeviceManager) UpdateDeviceStatus(id, status string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if device, exists := dm.devices[id]; exists {
		device.Status = status
		device.LastSeen = time.Now()
		device.UpdatedAt = time.Now()
	}
}

// EnableFileWatcher enables file watching for DHCP leases.
func (dm *DeviceManager) EnableFileWatcher(watchPaths []string) error {
	if dm.fileWatcher != nil {
		return customerrors.ErrFileWatcherAlreadyEnabled
	}

	// File watcher functionality is not implemented yet
	// This is a placeholder for future implementation
	return customerrors.ErrFileWatcherNotImplemented
}

// EnableFileWatcherAuto enables file watching with automatic DHCP lease file detection.
func (dm *DeviceManager) EnableFileWatcherAuto() error {
	// File watcher functionality is not implemented yet
	// This is a placeholder for future implementation
	return customerrors.ErrFileWatcherNotImplemented
}

// RefreshDHCPLeases refreshes DHCP leases from file.
func (dm *DeviceManager) RefreshDHCPLeases(ctx context.Context) error {
	if dm.leaseManager == nil {
		return customerrors.ErrLeaseManagerNotInitialized
	}

	if err := dm.leaseManager.LoadLeases(); err != nil {
		return fmt.Errorf("failed to load DHCP leases: %w", err)
	}

	// Reload devices from updated leases
	dm.loadFromDHCPLeases(ctx)

	return nil
}

// Close closes the device manager and cleans up resources.
func (dm *DeviceManager) Close() error {
	if dm.fileWatcher != nil {
		return dm.fileWatcher.Close()
	}

	return nil
}

func (dm *DeviceManager) loadFromDHCPLeases(ctx context.Context) []*Device {
	if dm == nil || dm.leaseManager == nil {
		return []*Device{}
	}

	// Load leases from file first
	if err := dm.leaseManager.LoadLeases(); err != nil {
		// Log error but continue - file might not exist yet
		// This is expected for OpenWrt and other systems
		zerolog.Ctx(ctx).Debug().
			Str("leases_path", dm.leaseManager.GetLeasesPath()).
			Err(err).
			Msg("Failed to load DHCP leases, file might not exist yet")

		return []*Device{}
	}

	leases := dm.leaseManager.GetAllLeases()

	zerolog.Ctx(ctx).Debug().
		Str("leases_path", dm.leaseManager.GetLeasesPath()).
		Int("leases_count", len(leases)).
		Msg("Loaded DHCP leases")

	devices := make([]*Device, 0, len(dm.devices))

	for _, lease := range leases {
		device := &Device{
			ID:        dm.generateDeviceID(),
			Name:      lease.Hostname,
			MAC:       lease.MAC,
			IP:        lease.IP,
			Hostname:  lease.Hostname,
			Status:    "online",
			Source:    "dhcp",
			Expire:    &lease.Expire,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			LastSeen:  time.Now(),
		}

		dm.updateDeviceCapabilities(device)
		dm.devices[device.ID] = device
		devices = append(devices, device)
	}

	return devices
}

func (dm *DeviceManager) updateFromScanResults(scanResults []*wol.ScanResult) []*Device {
	devices := make([]*Device, 0, len(dm.devices))

	for _, result := range scanResults {
		// Skip devices without MAC address
		if result.MAC == "" {
			continue
		}

		// Check if device already exists
		if existingDevice, exists := dm.GetDeviceByMAC(result.MAC); exists {
			// Update existing device
			existingDevice.IP = result.IP
			existingDevice.Hostname = result.Hostname
			existingDevice.Vendor = result.Vendor
			existingDevice.Status = "online"
			existingDevice.LastSeen = time.Now()
			existingDevice.UpdatedAt = time.Now()
			existingDevice.Source = "scan"

			dm.updateDeviceCapabilities(existingDevice)
			devices = append(devices, existingDevice)
		} else {
			// Create new device
			device := &Device{
				ID:        dm.generateDeviceID(),
				Name:      result.Hostname,
				MAC:       result.MAC,
				IP:        result.IP,
				Hostname:  result.Hostname,
				Vendor:    result.Vendor,
				Status:    "online",
				Source:    "scan",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				LastSeen:  time.Now(),
			}

			dm.updateDeviceCapabilities(device)
			dm.devices[device.ID] = device
			devices = append(devices, device)
		}
	}

	return devices
}

func (dm *DeviceManager) updateDeviceCapabilities(device *Device) {
	// Device capabilities are determined by methods
	// No need to set fields directly
}

func (dm *DeviceManager) generateDeviceID() string {
	return fmt.Sprintf("device_%d", time.Now().UnixNano())
}

// autoScanLoop is currently unused but kept for future implementation
// func (dm *DeviceManager) autoScanLoop() {
// 	ticker := time.NewTicker(dm.scanInterval)
// 	defer ticker.Stop()
//
// 	for range ticker.C {
// 		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
// 		_, err := dm.ScanDevices(ctx)
//
// 		cancel()
//
// 		if err != nil {
// 			zerolog.Ctx(context.Background()).Warn().Err(err).Msg("Auto-scan failed")
// 		}
// 	}
// }
