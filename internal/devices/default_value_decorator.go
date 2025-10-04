package devices

import (
	"context"
)

// DefaultValueDecorator sets default values for empty fields.
type DefaultValueDecorator struct{}

// NewDefaultValueDecorator creates a new default value decorator.
func NewDefaultValueDecorator() *DefaultValueDecorator {
	return &DefaultValueDecorator{}
}

// Decorate sets default values for empty fields.
func (d *DefaultValueDecorator) Decorate(ctx context.Context, devices []*Device) ([]*Device, error) {
	// Create a copy of devices to avoid mutating the input
	decorated := make([]*Device, len(devices))
	for i, device := range devices {
		// Use Clone method for efficient copying
		decorated[i] = device.Clone()

		// Set default values for empty fields
		if decorated[i].Status == "" {
			decorated[i].Status = StatusUnknown
		}

		if decorated[i].Vendor == "" {
			decorated[i].Vendor = "unknown" // Keep as string for vendor
		}
	}

	return decorated, nil
}
