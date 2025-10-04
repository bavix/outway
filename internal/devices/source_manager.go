package devices

import (
	"context"

	customerrors "github.com/bavix/outway/internal/errors"
)

// DeviceSourceManager manages multiple device discovery strategies.
type DeviceSourceManager struct {
	strategies []DeviceDiscoveryStrategy
	merger     DeviceMerger
	decorators []DeviceDecorator
}

// NewDeviceSourceManager creates a new device source manager.
func NewDeviceSourceManager() *DeviceSourceManager {
	return &DeviceSourceManager{
		strategies: make([]DeviceDiscoveryStrategy, 0),
		decorators: make([]DeviceDecorator, 0),
	}
}

// AddStrategy adds a discovery strategy.
func (dsm *DeviceSourceManager) AddStrategy(strategy DeviceDiscoveryStrategy) {
	dsm.strategies = append(dsm.strategies, strategy)
}

// AddDecorator adds a device decorator.
func (dsm *DeviceSourceManager) AddDecorator(decorator DeviceDecorator) {
	dsm.decorators = append(dsm.decorators, decorator)
}

// SetMerger sets the device merger.
func (dsm *DeviceSourceManager) SetMerger(merger DeviceMerger) {
	dsm.merger = merger
}

// DiscoverDevices discovers devices using all available strategies.
func (dsm *DeviceSourceManager) DiscoverDevices(ctx context.Context) ([]*Device, error) {
	var allDeviceInfos []*DeviceInfo

	// Discover devices using all strategies
	for _, strategy := range dsm.strategies {
		if !strategy.IsAvailable(ctx) {
			continue
		}

		devices, err := strategy.DiscoverDevices(ctx)
		if err != nil {
			// Log error but continue with other strategies
			continue
		}

		allDeviceInfos = append(allDeviceInfos, devices...)
	}

	// Merge device information
	if dsm.merger == nil {
		return nil, customerrors.ErrMergerNotSet
	}

	devices := dsm.merger.Merge(allDeviceInfos)

	// Apply decorators
	for _, decorator := range dsm.decorators {
		decorated, err := decorator.Decorate(ctx, devices)
		if err != nil {
			// Log error but continue
			continue
		}

		devices = decorated
	}

	return devices, nil
}
