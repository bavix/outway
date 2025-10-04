package wol

import (
	"context"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/rs/zerolog"

	customerrors "github.com/bavix/outway/internal/errors"
)

// NetworkInterface represents a network interface with broadcast information.
type NetworkInterface struct {
	Name         string   `json:"name"`
	Index        int      `json:"index"`
	MTU          int      `json:"mtu"`
	HardwareAddr string   `json:"hardware_addr"`
	IPs          []string `json:"ips"`
	Broadcast    string   `json:"broadcast"`
	IsUp         bool     `json:"is_up"`
	IsLoopback   bool     `json:"is_loopback"`
}

// InterfaceDetector detects network interfaces and calculates broadcast addresses.
type InterfaceDetector struct{}

// NewInterfaceDetector creates a new interface detector.
func NewInterfaceDetector() *InterfaceDetector {
	return &InterfaceDetector{}
}

// DetectInterfaces detects all available network interfaces.
func (d *InterfaceDetector) DetectInterfaces(ctx context.Context) ([]NetworkInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	result := make([]NetworkInterface, 0, len(interfaces))

	for _, iface := range interfaces {
		netIface, err := d.processInterface(iface)
		if err != nil {
			zerolog.Ctx(ctx).Warn().
				Str("interface", iface.Name).
				Err(err).
				Msg("failed to process interface")

			continue
		}

		// Skip loopback interfaces for WOL
		if netIface.IsLoopback {
			continue
		}

		result = append(result, netIface)
	}

	// Sort by name for consistent ordering
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result, nil
}

// GetBestInterface returns the best interface for Wake-on-LAN.
// Prefers non-loopback interfaces with broadcast addresses.
func (d *InterfaceDetector) GetBestInterface(ctx context.Context) (*NetworkInterface, error) {
	interfaces, err := d.DetectInterfaces(ctx)
	if err != nil {
		return nil, err
	}

	if len(interfaces) == 0 {
		return nil, customerrors.ErrNoSuitableNetworkInterfaces
	}

	// Find the first interface with a broadcast address
	for i := range interfaces {
		if interfaces[i].Broadcast != "" && interfaces[i].IsUp {
			return &interfaces[i], nil
		}
	}

	// Fallback to first available interface
	return &interfaces[0], nil
}

// GetInterfaceByName returns a specific interface by name.
func (d *InterfaceDetector) GetInterfaceByName(ctx context.Context, name string) (*NetworkInterface, error) {
	interfaces, err := d.DetectInterfaces(ctx)
	if err != nil {
		return nil, err
	}

	for i := range interfaces {
		if interfaces[i].Name == name {
			return &interfaces[i], nil
		}
	}

	return nil, customerrors.ErrInterfaceNotFoundWithName(name)
}

// ValidateInterface validates that an interface is suitable for Wake-on-LAN.
func (d *InterfaceDetector) ValidateInterface(iface *NetworkInterface) error {
	if iface == nil {
		return customerrors.ErrInterfaceIsNil
	}

	if iface.IsLoopback {
		return customerrors.ErrLoopbackInterfaceNotSuitable
	}

	if !iface.IsUp {
		return customerrors.ErrInterfaceIsDown
	}

	if iface.Broadcast == "" {
		return customerrors.ErrInterfaceNoBroadcastAddress
	}

	return nil
}

// GetBroadcastAddresses returns all available broadcast addresses.
func (d *InterfaceDetector) GetBroadcastAddresses(ctx context.Context) ([]string, error) {
	interfaces, err := d.DetectInterfaces(ctx)
	if err != nil {
		return nil, err
	}

	var broadcasts []string

	seen := make(map[string]bool)

	for _, iface := range interfaces {
		if iface.Broadcast != "" && !seen[iface.Broadcast] {
			broadcasts = append(broadcasts, iface.Broadcast)
			seen[iface.Broadcast] = true
		}
	}

	// Add common broadcast addresses
	commonBroadcasts := []string{
		"255.255.255.255", // Global broadcast
	}

	for _, addr := range commonBroadcasts {
		if !seen[addr] {
			broadcasts = append(broadcasts, addr)
		}
	}

	return broadcasts, nil
}

// IsValidBroadcastAddress checks if an address is a valid broadcast address.
func (d *InterfaceDetector) IsValidBroadcastAddress(ctx context.Context, addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}

	// Check if it's a valid IPv4 address
	if ip.To4() == nil {
		return false
	}

	// Check if it's a broadcast address
	// 255.255.255.255 is always valid
	if addr == "255.255.255.255" {
		return true
	}

	// Check if it's a network broadcast address
	interfaces, err := d.DetectInterfaces(ctx)
	if err != nil {
		return false
	}

	for _, iface := range interfaces {
		if iface.Broadcast == addr {
			return true
		}
	}

	return false
}

// GetInterfaceSummary returns a summary of all interfaces for logging.
func (d *InterfaceDetector) GetInterfaceSummary(ctx context.Context) string {
	interfaces, err := d.DetectInterfaces(ctx)
	if err != nil {
		return fmt.Sprintf("Error detecting interfaces: %v", err)
	}

	summary := make([]string, 0, len(interfaces))

	for _, iface := range interfaces {
		status := "down"
		if iface.IsUp {
			status = "up"
		}

		broadcast := "none"
		if iface.Broadcast != "" {
			broadcast = iface.Broadcast
		}

		summary = append(summary, fmt.Sprintf("%s(%s): %s [%s]",
			iface.Name, status, broadcast, strings.Join(iface.IPs, ",")))
	}

	return strings.Join(summary, "; ")
}

// processInterface processes a single network interface.
func (d *InterfaceDetector) processInterface(iface net.Interface) (NetworkInterface, error) {
	netIface := NetworkInterface{
		Name:         iface.Name,
		Index:        iface.Index,
		MTU:          iface.MTU,
		HardwareAddr: iface.HardwareAddr.String(),
		IsUp:         iface.Flags&net.FlagUp != 0,
		IsLoopback:   iface.Flags&net.FlagLoopback != 0,
	}

	// Get interface addresses
	addrs, err := iface.Addrs()
	if err != nil {
		return netIface, fmt.Errorf("failed to get interface addresses: %w", err)
	}

	var broadcastAddr string

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		// Only process IPv4 addresses for WOL
		if ipNet.IP.To4() == nil {
			continue
		}

		netIface.IPs = append(netIface.IPs, ipNet.IP.String())

		// Calculate broadcast address
		if broadcastAddr == "" {
			broadcastAddr = d.calculateBroadcast(ipNet)
		}
	}

	netIface.Broadcast = broadcastAddr

	return netIface, nil
}

// calculateBroadcast calculates the broadcast address for a given IP network.
func (d *InterfaceDetector) calculateBroadcast(ipNet *net.IPNet) string {
	if ipNet == nil {
		return ""
	}

	ip := ipNet.IP.To4()
	if ip == nil {
		return ""
	}

	mask := ipNet.Mask
	if len(mask) != ipv4AddressLength {
		return ""
	}

	// Calculate broadcast address
	broadcast := make(net.IP, ipv4AddressLength)
	for i := range ipv4AddressLength {
		broadcast[i] = ip[i] | (mask[i] ^ broadcastMask)
	}

	return broadcast.String()
}
