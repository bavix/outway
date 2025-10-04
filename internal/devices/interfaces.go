package devices

import (
	"context"
)

// DeviceSource represents the source of device information.
type DeviceSource string

// DeviceInfo represents basic device information from a source.
// This is an alias for Device to avoid duplication.
type DeviceInfo = Device

// DeviceDiscoveryStrategy defines the interface for device discovery strategies.
type DeviceDiscoveryStrategy interface {
	// Name returns the strategy name.
	Name() string

	// Priority returns the strategy priority (higher = more important).
	Priority() int

	// DiscoverDevices discovers devices using this strategy.
	DiscoverDevices(ctx context.Context) ([]*DeviceInfo, error)

	// IsAvailable checks if this strategy is available on the current system.
	IsAvailable(ctx context.Context) bool
}

// DeviceMerger defines the interface for merging device information.
type DeviceMerger interface {
	// Merge merges device information from multiple sources.
	Merge(devices []*DeviceInfo) []*Device
}

// DeviceDecorator defines the interface for decorating devices with additional information.
type DeviceDecorator interface {
	// Decorate adds additional information to devices and returns the decorated devices.
	// This is a functional approach that doesn't mutate the input.
	Decorate(ctx context.Context, devices []*Device) ([]*Device, error)
}
