package wol

import (
	"fmt"
	"sync"
	"time"

	customerrors "github.com/bavix/outway/internal/errors"
)

// StoredDevice represents a device stored in memory.
type StoredDevice struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	MAC       string    `json:"mac"`
	IP        string    `json:"ip"`
	Hostname  string    `json:"hostname,omitempty"`
	Vendor    string    `json:"vendor,omitempty"`
	Status    string    `json:"status"` // online, offline, unknown
	LastSeen  time.Time `json:"last_seen"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeviceStorage manages in-memory storage of WOL devices.
type DeviceStorage struct {
	devices map[string]*StoredDevice
	mu      sync.RWMutex
}

// NewDeviceStorage creates a new device storage manager.
func NewDeviceStorage() *DeviceStorage {
	return &DeviceStorage{
		devices: make(map[string]*StoredDevice),
	}
}

// Load initializes empty storage (no-op for in-memory storage).
func (ds *DeviceStorage) Load() error {
	// No-op for in-memory storage
	return nil
}

// AddDevice adds a new device to storage.
func (ds *DeviceStorage) AddDevice(device *StoredDevice) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if device.ID == "" {
		device.ID = generateDeviceID()
	}

	now := time.Now()
	if device.CreatedAt.IsZero() {
		device.CreatedAt = now
	}

	device.UpdatedAt = now

	ds.devices[device.ID] = device

	return nil
}

// UpdateDevice updates an existing device.
func (ds *DeviceStorage) UpdateDevice(device *StoredDevice) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if _, exists := ds.devices[device.ID]; !exists {
		return customerrors.ErrDeviceNotFoundWithID(device.ID)
	}

	device.UpdatedAt = time.Now()
	ds.devices[device.ID] = device

	return nil
}

// GetDevice retrieves a device by ID.
func (ds *DeviceStorage) GetDevice(id string) (*StoredDevice, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	device, exists := ds.devices[id]

	return device, exists
}

// GetDeviceByMAC retrieves a device by MAC address.
func (ds *DeviceStorage) GetDeviceByMAC(mac string) (*StoredDevice, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	for _, device := range ds.devices {
		if device.MAC == mac {
			return device, true
		}
	}

	return nil, false
}

// GetAllDevices returns all stored devices.
func (ds *DeviceStorage) GetAllDevices() []*StoredDevice {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	devices := make([]*StoredDevice, 0, len(ds.devices))
	for _, device := range ds.devices {
		devices = append(devices, device)
	}

	return devices
}

// DeleteDevice removes a device from storage.
func (ds *DeviceStorage) DeleteDevice(id string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if _, exists := ds.devices[id]; !exists {
		return customerrors.ErrDeviceNotFoundWithID(id)
	}

	delete(ds.devices, id)

	return nil
}

// UpdateDeviceStatus updates device status and last seen time.
func (ds *DeviceStorage) UpdateDeviceStatus(id, status string) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	device, exists := ds.devices[id]
	if !exists {
		return customerrors.ErrDeviceNotFoundWithID(id)
	}

	device.Status = status
	device.LastSeen = time.Now()
	device.UpdatedAt = time.Now()

	return nil
}

// UpdateDeviceFromScan updates device info from network scan.
func (ds *DeviceStorage) UpdateDeviceFromScan(mac, ip, hostname, vendor string) (*StoredDevice, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Check if device exists by MAC
	for _, device := range ds.devices {
		if device.MAC == mac {
			// Update existing device
			device.IP = ip
			device.Hostname = hostname
			device.Vendor = vendor
			device.Status = "online"
			device.LastSeen = time.Now()
			device.UpdatedAt = time.Now()

			return device, nil
		}
	}

	// Create new device
	device := &StoredDevice{
		ID:        generateDeviceID(),
		Name:      hostname,
		MAC:       mac,
		IP:        ip,
		Hostname:  hostname,
		Vendor:    vendor,
		Status:    "online",
		LastSeen:  time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ds.devices[device.ID] = device

	return device, nil
}

// generateDeviceID generates a unique device ID.
func generateDeviceID() string {
	return fmt.Sprintf("device_%d", time.Now().UnixNano())
}
