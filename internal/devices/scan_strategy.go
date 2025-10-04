package devices

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/bavix/outway/internal/wol"
)

// NetworkScanStrategy discovers devices through network scanning.
type NetworkScanStrategy struct {
	scanner  *wol.NetworkScanner
	priority int
}

// NewNetworkScanStrategy creates a new network scan strategy.
func NewNetworkScanStrategy() *NetworkScanStrategy {
	return &NetworkScanStrategy{
		scanner:  wol.NewNetworkScanner(),
		priority: PriorityNetworkScan, // Lower priority than DHCP
	}
}

// Name returns the strategy name.
func (n *NetworkScanStrategy) Name() string {
	return "network-scan"
}

// Priority returns the strategy priority.
func (n *NetworkScanStrategy) Priority() int {
	return n.priority
}

// IsAvailable checks if network scanning is available.
func (n *NetworkScanStrategy) IsAvailable(ctx context.Context) bool {
	// Check if we can get local network CIDR
	_, err := n.scanner.GetLocalNetworkCIDR()

	return err == nil
}

// DiscoverDevices discovers devices through network scanning.
func (n *NetworkScanStrategy) DiscoverDevices(ctx context.Context) ([]*DeviceInfo, error) {
	logger := zerolog.Ctx(ctx)

	// Get local network CIDR
	networkCIDR, err := n.scanner.GetLocalNetworkCIDR()
	if err != nil {
		logger.Debug().
			Err(err).
			Msg("Failed to get local network CIDR for scanning")

		return []*DeviceInfo{}, nil
	}

	// Perform network scan
	scanResults, err := n.scanner.ScanNetwork(ctx, networkCIDR)
	if err != nil {
		logger.Debug().
			Str("network", networkCIDR).
			Err(err).
			Msg("Network scan failed")

		return []*DeviceInfo{}, nil
	}

	devices := make([]*DeviceInfo, 0, len(scanResults))

	logger.Debug().
		Str("network", networkCIDR).
		Int("devices_found", len(scanResults)).
		Msg("Discovered devices through network scanning")

	for _, result := range scanResults {
		device := &DeviceInfo{
			MAC:      result.MAC,
			IP:       result.IP,
			Hostname: result.Hostname,
			Vendor:   result.Vendor,
			Status:   StatusOnline, // If device responds to scan, it's online
			Source:   SourceNetworkScan,
			LastSeen: time.Now(),
			Expire:   nil, // Scan results don't have expiration
		}

		devices = append(devices, device)
	}

	return devices, nil
}

// GetScanner returns the network scanner for external use.
func (n *NetworkScanStrategy) GetScanner() *wol.NetworkScanner {
	return n.scanner
}
