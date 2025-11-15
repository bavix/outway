package lanresolver

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"

	"github.com/bavix/outway/internal/errors"
	"github.com/bavix/outway/internal/localzone"
	"github.com/bavix/outway/internal/wol"
)

const (
	defaultScanTimeout = 10 * time.Second
)

// WOLResolver extends LANResolver with Wake-on-LAN capabilities.
type WOLResolver struct {
	*LANResolver

	wolService *wol.WakeOnLan
	storage    *wol.DeviceStorage
	scanner    *wol.NetworkScanner
	mu         sync.RWMutex
}

// NewWOLResolver creates a new WOL-enabled LAN resolver.
func NewWOLResolver(
	next interface {
		Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error)
	},
	zoneDetector *localzone.ZoneDetector,
	leaseManager *LeaseManager,
) *WOLResolver {
	lanResolver := NewLANResolver(next, zoneDetector, leaseManager)
	wolService := wol.NewWakeOnLan()
	storage := wol.NewDeviceStorage()
	scanner := wol.NewNetworkScanner()

	// Load existing devices
	if err := storage.Load(); err != nil {
		zerolog.Ctx(context.Background()).Warn().Err(err).Msg("Failed to load WOL devices storage")
	}

	return &WOLResolver{
		LANResolver: lanResolver,
		wolService:  wolService,
		storage:     storage,
		scanner:     scanner,
	}
}

// NewWOLResolverWithConfig creates a new WOL-enabled LAN resolver with custom WOL configuration.
func NewWOLResolverWithConfig(
	next interface {
		Resolve(ctx context.Context, q *dns.Msg) (*dns.Msg, string, error)
	},
	zoneDetector *localzone.ZoneDetector,
	leaseManager *LeaseManager,
	wolConfig *wol.Config,
) *WOLResolver {
	lanResolver := NewLANResolver(next, zoneDetector, leaseManager)
	wolService := wol.NewWakeOnLanWithConfig(wolConfig)
	storage := wol.NewDeviceStorage()
	scanner := wol.NewNetworkScanner()

	// Load existing devices
	if err := storage.Load(); err != nil {
		zerolog.Ctx(context.Background()).Warn().Err(err).Msg("Failed to load WOL devices storage")
	}

	return &WOLResolver{
		LANResolver: lanResolver,
		wolService:  wolService,
		storage:     storage,
		scanner:     scanner,
	}
}

// GetWOLService returns the Wake-on-LAN service.
func (wr *WOLResolver) GetWOLService() *wol.WakeOnLan {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	return wr.wolService
}

// GetWOLInterfaces returns available network interfaces for Wake-on-LAN.
func (wr *WOLResolver) GetWOLInterfaces(ctx context.Context) ([]wol.NetworkInterface, error) {
	return wr.wolService.GetInterfaces(ctx)
}

// GetWOLBroadcastAddresses returns available broadcast addresses for Wake-on-LAN.
func (wr *WOLResolver) GetWOLBroadcastAddresses(ctx context.Context) ([]string, error) {
	return wr.wolService.GetBroadcastAddresses(ctx)
}

// GetWOLConfig returns the current Wake-on-LAN configuration.
func (wr *WOLResolver) GetWOLConfig() *wol.Config {
	return wr.wolService.GetConfig()
}

// SetWOLConfig updates the Wake-on-LAN configuration.
func (wr *WOLResolver) SetWOLConfig(config *wol.Config) error {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	return wr.wolService.SetConfig(config)
}

// UpdateWOLConfig updates specific Wake-on-LAN configuration fields.
func (wr *WOLResolver) UpdateWOLConfig(updates map[string]any) error {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	return wr.wolService.UpdateConfig(updates)
}

// SendWOLPacket sends a Wake-on-LAN packet.
func (wr *WOLResolver) SendWOLPacket(ctx context.Context, req *wol.WakeOnLanRequest) (*wol.WakeOnLanResponse, error) {
	return wr.wolService.SendMagicPacket(ctx, req)
}

// SendWOLPacketToInterface sends a Wake-on-LAN packet to a specific interface.
func (wr *WOLResolver) SendWOLPacketToInterface(
	ctx context.Context,
	req *wol.WakeOnLanRequest,
	interfaceName string,
) (*wol.WakeOnLanResponse, error) {
	return wr.wolService.SendMagicPacketToInterface(ctx, req, interfaceName)
}

// SendWOLPacketToAllInterfaces sends a Wake-on-LAN packet to all suitable interfaces.
func (wr *WOLResolver) SendWOLPacketToAllInterfaces(ctx context.Context, req *wol.WakeOnLanRequest) ([]wol.WakeOnLanResponse, error) {
	return wr.wolService.SendMagicPacketToAllInterfaces(ctx, req)
}

// SendWOLPacketWithRetry sends a Wake-on-LAN packet with retry logic.
func (wr *WOLResolver) SendWOLPacketWithRetry(ctx context.Context, req *wol.WakeOnLanRequest) (*wol.WakeOnLanResponse, error) {
	return wr.wolService.SendMagicPacketWithRetry(ctx, req)
}

// ValidateWOLMAC validates a MAC address for Wake-on-LAN.
func (wr *WOLResolver) ValidateWOLMAC(mac string) error {
	return wr.wolService.ValidateMAC(mac)
}

// GetWOLStatus returns the Wake-on-LAN service status.
func (wr *WOLResolver) GetWOLStatus(ctx context.Context) map[string]any {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	logger := zerolog.Ctx(ctx)
	config := wr.wolService.GetConfig()

	interfaces, err := wr.wolService.GetInterfaces(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("failed to get interfaces for WOL status")

		interfaces = []wol.NetworkInterface{}
	}

	validInterfaces := 0

	for range interfaces {
		if wr.wolService.ValidateMAC("00:00:00:00:00:00") == nil { // Dummy validation
			validInterfaces++
		}
	}

	return map[string]any{
		"enabled":          wr.wolService.IsEnabled(),
		"config":           config,
		"interfaces_count": len(interfaces),
		"valid_interfaces": validInterfaces,
	}
}

// GetWOLDevices returns devices that can be woken up (from storage + DHCP leases).
func (wr *WOLResolver) GetWOLDevices() []WOLDevice {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	// Get devices from storage
	storedDevices := wr.storage.GetAllDevices()
	devices := make([]WOLDevice, 0, len(storedDevices))

	for _, stored := range storedDevices {
		device := WOLDevice{
			Hostname: stored.Hostname,
			IP:       stored.IP,
			MAC:      stored.MAC,
			Expire:   stored.LastSeen.Format("2006-01-02 15:04:05"),
			ID:       stored.ID,
		}

		// Validate MAC address for WOL
		if wr.wolService.ValidateMAC(stored.MAC) == nil {
			device.CanWake = true
		}

		devices = append(devices, device)
	}

	// Also include devices from DHCP leases that are not in storage
	leases := wr.LeaseManager.GetAllLeases()
	for _, lease := range leases {
		// Check if device already exists in storage
		exists := false

		for _, stored := range storedDevices {
			if stored.MAC == lease.MAC {
				exists = true

				break
			}
		}

		if !exists {
			device := WOLDevice{
				Hostname: lease.Hostname,
				IP:       lease.IP,
				MAC:      lease.MAC,
				Expire:   lease.Expire.Format("2006-01-02 15:04:05"),
				ID:       lease.ID,
			}

			// Validate MAC address for WOL
			if wr.wolService.ValidateMAC(lease.MAC) == nil {
				device.CanWake = true
			}

			devices = append(devices, device)
		}
	}

	return devices
}

// WakeDevice wakes up a device by hostname or MAC address.
func (wr *WOLResolver) WakeDevice(ctx context.Context, identifier string, interfaceName string) (*wol.WakeOnLanResponse, error) {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	// Try to find device by hostname first
	lease := wr.LeaseManager.GetLease(identifier)
	if lease == nil {
		// Try to find by MAC address
		leases := wr.LeaseManager.GetAllLeases()
		for _, l := range leases {
			if strings.EqualFold(l.MAC, identifier) {
				lease = l

				break
			}
		}
	}

	if lease == nil {
		return &wol.WakeOnLanResponse{
			Success: false,
			Message: "device not found",
		}, nil
	}

	// Validate MAC address
	if err := wr.wolService.ValidateMAC(lease.MAC); err != nil {
		return &wol.WakeOnLanResponse{
			Success: false,
			Message: "invalid MAC address: " + err.Error(),
			MAC:     lease.MAC,
		}, nil
	}

	// Create WOL request
	req := &wol.WakeOnLanRequest{
		MAC: lease.MAC,
	}

	// Send WOL packet
	if interfaceName != "" {
		return wr.wolService.SendMagicPacketToInterface(ctx, req, interfaceName)
	}

	return wr.wolService.SendMagicPacket(ctx, req)
}

// WakeAllDevices wakes up all devices that can be woken up.
func (wr *WOLResolver) WakeAllDevices(ctx context.Context, interfaceName string) ([]wol.WakeOnLanResponse, error) {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	devices := wr.GetWOLDevices()

	responses := make([]wol.WakeOnLanResponse, 0, len(devices))

	for _, device := range devices {
		if !device.CanWake {
			responses = append(responses, wol.WakeOnLanResponse{
				Success: false,
				Message: "invalid MAC address",
				MAC:     device.MAC,
			})

			continue
		}

		req := &wol.WakeOnLanRequest{
			MAC: device.MAC,
		}

		var (
			resp *wol.WakeOnLanResponse
			err  error
		)

		if interfaceName != "" {
			resp, err = wr.wolService.SendMagicPacketToInterface(ctx, req, interfaceName)
		} else {
			resp, err = wr.wolService.SendMagicPacket(ctx, req)
		}

		if err != nil {
			resp = &wol.WakeOnLanResponse{
				Success: false,
				Message: err.Error(),
				MAC:     device.MAC,
			}
		}

		responses = append(responses, *resp)
	}

	return responses, nil
}

// WOLDevice represents a device that can be woken up.
type WOLDevice struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Expire   string `json:"expire"`
	ID       string `json:"id"`
	CanWake  bool   `json:"can_wake"`
}

// GetWOLDeviceByHostname returns a WOL device by hostname.
func (wr *WOLResolver) GetWOLDeviceByHostname(hostname string) *WOLDevice {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	lease := wr.LeaseManager.GetLease(hostname)
	if lease == nil {
		return nil
	}

	device := WOLDevice{
		Hostname: lease.Hostname,
		IP:       lease.IP,
		MAC:      lease.MAC,
		Expire:   lease.Expire.Format("2006-01-02 15:04:05"),
		ID:       lease.ID,
	}

	// Validate MAC address for WOL
	if wr.wolService.ValidateMAC(lease.MAC) == nil {
		device.CanWake = true
	}

	return &device
}

// AddWOLDevice adds a new device to storage.
func (wr *WOLResolver) AddWOLDevice(name, mac, ip, hostname, vendor string) (*wol.StoredDevice, error) {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	device := &wol.StoredDevice{
		Name:     name,
		MAC:      mac,
		IP:       ip,
		Hostname: hostname,
		Vendor:   vendor,
		Status:   "unknown",
	}

	return device, wr.storage.AddDevice(device)
}

// UpdateWOLDevice updates an existing device in storage.
func (wr *WOLResolver) UpdateWOLDevice(id, name, mac, ip, hostname, vendor, status string) error {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	device, exists := wr.storage.GetDevice(id)
	if !exists {
		return errors.ErrDeviceNotFoundWithID(id)
	}

	device.Name = name
	device.MAC = mac
	device.IP = ip
	device.Hostname = hostname
	device.Vendor = vendor
	device.Status = status

	return wr.storage.UpdateDevice(device)
}

// DeleteWOLDevice removes a device from storage.
func (wr *WOLResolver) DeleteWOLDevice(id string) error {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	return wr.storage.DeleteDevice(id)
}

// ScanAndUpdateDevices scans the network and updates device information.
//
//nolint:funlen
func (wr *WOLResolver) ScanAndUpdateDevices(ctx context.Context) ([]*wol.StoredDevice, error) {
	wr.mu.Lock()
	defer wr.mu.Unlock()

	logger := zerolog.Ctx(ctx)

	// Get local network CIDR
	networkCIDR, err := wr.scanner.GetLocalNetworkCIDR()
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to get local network, falling back to DHCP leases")

		return wr.scanFromDHCPLeases(ctx), nil
	}

	logger.Info().Str("detected_network", networkCIDR).Msg("Detected local network")

	logger.Info().
		Str("network", networkCIDR).
		Msg("Starting network scan")

		// Perform network scan with timeout
	scanCtx, cancel := context.WithTimeout(ctx, defaultScanTimeout)
	defer cancel()

	scanResults, err := wr.scanner.ScanNetwork(scanCtx, networkCIDR)
	if err != nil {
		logger.Warn().Err(err).Msg("Network scan failed, falling back to DHCP leases")

		return wr.scanFromDHCPLeases(ctx), nil
	}

	// Update storage with scan results
	updatedDevices := make([]*wol.StoredDevice, 0, len(scanResults))

	logger.Info().
		Int("scan_results_count", len(scanResults)).
		Msg("Processing scan results")

	for _, result := range scanResults {
		// Skip devices without MAC address
		if result.MAC == "" {
			logger.Debug().
				Str("ip", result.IP).
				Msg("Skipping device without MAC address")

			continue
		}

		logger.Debug().
			Str("mac", result.MAC).
			Str("ip", result.IP).
			Str("hostname", result.Hostname).
			Msg("Processing device from scan")

		// Update or create device from scan result
		device, err := wr.storage.UpdateDeviceFromScan(
			result.MAC,
			result.IP,
			result.Hostname,
			result.Vendor,
		)
		if err != nil {
			logger.Warn().
				Str("mac", result.MAC).
				Str("ip", result.IP).
				Err(err).
				Msg("Failed to update device from scan")

			continue
		}

		logger.Debug().
			Str("device_id", device.ID).
			Str("mac", device.MAC).
			Str("ip", device.IP).
			Msg("Device updated in storage")

		updatedDevices = append(updatedDevices, device)
	}

	// Also include devices from DHCP leases that weren't found in scan
	wr.mergeDHCPLeases(ctx, updatedDevices)

	logger.Info().
		Int("devices_found", len(updatedDevices)).
		Msg("Network scan completed")

	return updatedDevices, nil
}

// GetWOLDeviceByMAC returns a WOL device by MAC address.
func (wr *WOLResolver) GetWOLDeviceByMAC(mac string) *WOLDevice {
	wr.mu.RLock()
	defer wr.mu.RUnlock()

	leases := wr.LeaseManager.GetAllLeases()
	for _, lease := range leases {
		if strings.EqualFold(lease.MAC, mac) {
			device := WOLDevice{
				Hostname: lease.Hostname,
				IP:       lease.IP,
				MAC:      lease.MAC,
				Expire:   lease.Expire.Format("2006-01-02 15:04:05"),
				ID:       lease.ID,
			}

			// Validate MAC address for WOL
			if wr.wolService.ValidateMAC(lease.MAC) == nil {
				device.CanWake = true
			}

			return &device
		}
	}

	return nil
}

// scanFromDHCPLeases scans devices from DHCP leases only.
func (wr *WOLResolver) scanFromDHCPLeases(ctx context.Context) []*wol.StoredDevice {
	logger := zerolog.Ctx(ctx)

	// Get current DHCP leases
	leases := wr.LeaseManager.GetAllLeases()
	updatedDevices := make([]*wol.StoredDevice, 0, len(leases))

	for _, lease := range leases {
		// Update or create device from lease
		device, err := wr.storage.UpdateDeviceFromScan(
			lease.MAC,
			lease.IP,
			lease.Hostname,
			"", // Vendor detection would need additional implementation
		)
		if err != nil {
			logger.Warn().
				Str("mac", lease.MAC).
				Err(err).
				Msg("Failed to update device from DHCP lease")

			continue
		}

		updatedDevices = append(updatedDevices, device)
	}

	return updatedDevices
}

// mergeDHCPLeases merges DHCP leases with scan results.
func (wr *WOLResolver) mergeDHCPLeases(ctx context.Context, scanDevices []*wol.StoredDevice) {
	logger := zerolog.Ctx(ctx)

	// Create a map of MAC addresses from scan results
	scanMACs := make(map[string]bool)
	for _, device := range scanDevices {
		scanMACs[device.MAC] = true
	}

	// Get DHCP leases
	leases := wr.LeaseManager.GetAllLeases()

	for _, lease := range leases {
		// Skip if already found in scan
		if scanMACs[lease.MAC] {
			continue
		}

		// Add DHCP lease device
		device, err := wr.storage.UpdateDeviceFromScan(
			lease.MAC,
			lease.IP,
			lease.Hostname,
			"",
		)
		if err != nil {
			logger.Warn().
				Str("mac", lease.MAC).
				Err(err).
				Msg("Failed to add DHCP lease device")

			continue
		}

		scanDevices = append(scanDevices, device)
	}
}
