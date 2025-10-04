package devices

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"

	"github.com/bavix/outway/internal/lanresolver"
)

// DHCPLeaseStrategy discovers devices from DHCP lease files.
type DHCPLeaseStrategy struct {
	leaseManager *lanresolver.LeaseManager
	priority     int
}

// NewDHCPLeaseStrategy creates a new DHCP lease strategy.
func NewDHCPLeaseStrategy() *DHCPLeaseStrategy {
	paths := GetCommonLeasePaths()

	var leaseManager *lanresolver.LeaseManager

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			leaseManager = lanresolver.NewLeaseManager(path)

			break
		}
	}

	// Return nil if no DHCP lease file found
	if leaseManager == nil {
		return nil
	}

	return &DHCPLeaseStrategy{
		leaseManager: leaseManager,
		priority:     PriorityDHCPLeases, // Highest priority - DHCP is the source of truth
	}
}

// Name returns the strategy name.
func (d *DHCPLeaseStrategy) Name() string {
	return "dhcp-leases"
}

// Priority returns the strategy priority.
func (d *DHCPLeaseStrategy) Priority() int {
	return d.priority
}

// IsAvailable checks if DHCP lease files are available.
func (d *DHCPLeaseStrategy) IsAvailable(ctx context.Context) bool {
	// Check if the lease file exists and is readable
	path := d.leaseManager.GetLeasesPath()
	if _, err := os.Stat(path); err != nil {
		return false
	}

	// Try to read the file
	// #nosec G304 - We need to read DHCP lease files from various system paths
	if _, err := os.ReadFile(path); err != nil {
		return false
	}

	return true
}

// DiscoverDevices discovers devices from DHCP leases.
func (d *DHCPLeaseStrategy) DiscoverDevices(ctx context.Context) ([]*DeviceInfo, error) {
	logger := zerolog.Ctx(ctx)

	// Load leases from file
	if err := d.leaseManager.LoadLeases(); err != nil {
		logger.Debug().
			Str("leases_path", d.leaseManager.GetLeasesPath()).
			Err(err).
			Msg("Failed to load DHCP leases")

		return []*DeviceInfo{}, nil
	}

	leases := d.leaseManager.GetAllLeases()
	devices := make([]*DeviceInfo, 0, len(leases))

	logger.Debug().
		Str("leases_path", d.leaseManager.GetLeasesPath()).
		Int("leases_count", len(leases)).
		Msg("Discovered devices from DHCP leases")

	for _, lease := range leases {
		device := &DeviceInfo{
			MAC:      lease.MAC,
			IP:       lease.IP,
			Hostname: lease.Hostname,
			Vendor:   "",            // DHCP doesn't provide vendor info
			Status:   StatusUnknown, // DHCP doesn't know if device is online
			Source:   SourceDHCPLeases,
			LastSeen: time.Now(),
			Expire:   &lease.Expire,
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// GetLeaseManager returns the lease manager for external use.
func (d *DHCPLeaseStrategy) GetLeaseManager() *lanresolver.LeaseManager {
	return d.leaseManager
}

// RefreshLeases refreshes the DHCP leases.
func (d *DHCPLeaseStrategy) RefreshLeases() error {
	return d.leaseManager.LoadLeases()
}

// SetLeaseFile sets a custom lease file path.
func (d *DHCPLeaseStrategy) SetLeaseFile(path string) {
	d.leaseManager.SetLeaseFile(path)
}
