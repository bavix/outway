package lanresolver

import (
	"context"

	"github.com/bavix/outway/internal/wol"
)

// WOLConfigInterface handles WOL configuration operations.
type WOLConfigInterface interface {
	GetWOLConfig() *wol.Config
	SetWOLConfig(config *wol.Config) error
	UpdateWOLConfig(updates map[string]any) error
}

// WOLDeviceInterface handles WOL device operations.
type WOLDeviceInterface interface {
	GetWOLDevices() []WOLDevice
	GetWOLDeviceByHostname(hostname string) *WOLDevice
	GetWOLDeviceByMAC(mac string) *WOLDevice
	AddWOLDevice(name, mac, ip, hostname, vendor string) (*wol.StoredDevice, error)
	UpdateWOLDevice(id, name, mac, ip, hostname, vendor, status string) error
	DeleteWOLDevice(id string) error
	ScanAndUpdateDevices(ctx context.Context) ([]*wol.StoredDevice, error)
}

// WOLNetworkInterface handles WOL network operations.
type WOLNetworkInterface interface {
	GetWOLInterfaces(ctx context.Context) ([]wol.NetworkInterface, error)
	GetWOLBroadcastAddresses(ctx context.Context) ([]string, error)
}

// WOLWakeInterface handles WOL wake operations.
type WOLWakeInterface interface {
	WakeDevice(ctx context.Context, identifier string, interfaceName string) (*wol.WakeOnLanResponse, error)
	WakeAllDevices(ctx context.Context, interfaceName string) ([]wol.WakeOnLanResponse, error)
	SendWOLPacket(ctx context.Context, req *wol.WakeOnLanRequest) (*wol.WakeOnLanResponse, error)
	SendWOLPacketToInterface(ctx context.Context, req *wol.WakeOnLanRequest, interfaceName string) (*wol.WakeOnLanResponse, error)
	SendWOLPacketToAllInterfaces(ctx context.Context, req *wol.WakeOnLanRequest) ([]wol.WakeOnLanResponse, error)
	SendWOLPacketWithRetry(ctx context.Context, req *wol.WakeOnLanRequest) (*wol.WakeOnLanResponse, error)
	ValidateWOLMAC(mac string) error
}

// WOLResolverInterface combines all WOL interfaces.
type WOLResolverInterface interface {
	WOLConfigInterface
	WOLDeviceInterface
	WOLNetworkInterface
	WOLWakeInterface
	GetWOLStatus(ctx context.Context) map[string]any
}
