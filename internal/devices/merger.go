package devices

import (
	"sort"
	"strings"
	"time"
)

// DefaultDeviceMerger is the default device merger implementation.
type DefaultDeviceMerger struct{}

// NewDefaultDeviceMerger creates a new default device merger.
func NewDefaultDeviceMerger() *DefaultDeviceMerger {
	return &DefaultDeviceMerger{}
}

// Merge merges device information from multiple sources.
// DHCP leases have the highest priority for MAC and hostname.
func (m *DefaultDeviceMerger) Merge(deviceInfos []*DeviceInfo) []*Device {
	// Group devices by MAC address
	deviceMap := make(map[string]*Device)

	// Sort by priority (higher priority first)
	sort.Slice(deviceInfos, func(i, j int) bool {
		return deviceInfos[i].Source == SourceDHCPLeases && deviceInfos[j].Source != SourceDHCPLeases
	})

	for _, info := range deviceInfos {
		if info.MAC == "" {
			continue // Skip devices without MAC
		}

		// Normalize MAC address
		mac := strings.ToLower(strings.ReplaceAll(info.MAC, ":", ""))

		device, exists := deviceMap[mac]
		if !exists {
			// Create new device
			device = &Device{
				ID:        generateDeviceID(),
				MAC:       info.MAC,
				IP:        info.IP,
				Hostname:  info.Hostname,
				Vendor:    info.Vendor,
				Status:    info.Status,
				Source:    info.Source,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				LastSeen:  info.LastSeen,
			}

			if info.Expire != nil {
				device.Expire = info.Expire
			}

			deviceMap[mac] = device
		} else {
			// Merge information, prioritizing DHCP leases
			m.mergeDeviceInfo(device, info)
		}
	}

	// Convert map to slice
	devices := make([]*Device, 0, len(deviceMap))
	for _, device := range deviceMap {
		devices = append(devices, device)
	}

	return devices
}

// mergeDeviceInfo merges device information, prioritizing DHCP leases.
func (m *DefaultDeviceMerger) mergeDeviceInfo(device *Device, info *DeviceInfo) *Device {
	// Update basic fields
	m.updateBasicFields(device, info)

	// Update source-specific fields
	m.updateSourceSpecificFields(device, info)

	// Update vendor and status
	m.updateVendorAndStatus(device, info)

	// Update source to reflect multiple sources
	m.updateSource(device, info)

	return device
}

// updateBasicFields updates basic device fields.
func (m *DefaultDeviceMerger) updateBasicFields(device *Device, info *DeviceInfo) {
	device.UpdatedAt = time.Now()

	if info.LastSeen.After(device.LastSeen) {
		device.LastSeen = info.LastSeen
	}
}

// updateSourceSpecificFields updates fields based on source priority.
func (m *DefaultDeviceMerger) updateSourceSpecificFields(device *Device, info *DeviceInfo) {
	if info.Source == SourceDHCPLeases {
		m.updateFromDHCP(device, info)
	} else {
		m.updateFromOtherSource(device, info)
	}
}

// updateFromDHCP updates device from DHCP source.
func (m *DefaultDeviceMerger) updateFromDHCP(device *Device, info *DeviceInfo) {
	if info.Hostname != "" {
		device.Hostname = info.Hostname
		device.Name = info.Hostname
	}

	if info.IP != "" {
		device.IP = info.IP
	}

	if info.Expire != nil {
		device.Expire = info.Expire
	}

	device.Source = info.Source
}

// updateFromOtherSource updates device from non-DHCP source.
func (m *DefaultDeviceMerger) updateFromOtherSource(device *Device, info *DeviceInfo) {
	if device.Hostname == "" && info.Hostname != "" {
		device.Hostname = info.Hostname
		device.Name = info.Hostname
	}

	if device.IP == "" && info.IP != "" {
		device.IP = info.IP
	}
}

// updateVendorAndStatus updates vendor and status fields.
func (m *DefaultDeviceMerger) updateVendorAndStatus(device *Device, info *DeviceInfo) {
	if info.Vendor != "" {
		device.Vendor = info.Vendor
	}

	if info.Status == StatusOnline || device.Status == StatusUnknown {
		device.Status = info.Status
	}
}

// updateSource updates source field to reflect multiple sources.
func (m *DefaultDeviceMerger) updateSource(device *Device, info *DeviceInfo) {
	if device.Source != info.Source {
		device.Source = SourceMultiple
	}
}

// generateDeviceID generates a unique device ID.
func generateDeviceID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(RandomIDLength)
}

// randomString generates a random string of specified length.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}

	return string(b)
}
