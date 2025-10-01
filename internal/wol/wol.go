package wol

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
)

var (
	// ErrInvalidMAC is returned when a MAC address is invalid.
	ErrInvalidMAC = errors.New("invalid MAC address")
	// ErrNoInterfaces is returned when no network interfaces are found.
	ErrNoInterfaces = errors.New("no network interfaces found")
)

const (
	// DefaultPort is the default UDP port for Wake-on-LAN packets.
	DefaultPort = 9
	// MagicPacketSize is the size of a Wake-on-LAN magic packet (6 + 16*6 = 102 bytes).
	MagicPacketSize = 102
)

// NetworkInterface represents a network interface with its broadcast address.
type NetworkInterface struct {
	Name      string `json:"name"`
	IP        string `json:"ip"`
	Broadcast string `json:"broadcast"`
	Network   string `json:"network"`
}

// Config holds Wake-on-LAN configuration.
type Config struct {
	DefaultPort int `json:"default_port"`
}

// Client handles Wake-on-LAN operations.
type Client struct {
	config *Config
}

// NewClient creates a new WOL client.
func NewClient(config *Config) *Client {
	if config == nil {
		config = &Config{DefaultPort: DefaultPort}
	}
	if config.DefaultPort == 0 {
		config.DefaultPort = DefaultPort
	}
	return &Client{config: config}
}

// GetNetworkInterfaces returns all network interfaces with their broadcast addresses.
func (c *Client) GetNetworkInterfaces() ([]NetworkInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var result []NetworkInterface

	for _, iface := range ifaces {
		// Skip loopback interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Skip interfaces that are down
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Skip interfaces that don't support broadcast
		if iface.Flags&net.FlagBroadcast == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			// Skip IPv6 addresses
			if ipNet.IP.To4() == nil {
				continue
			}

			// Calculate broadcast address
			broadcast := calculateBroadcast(ipNet)

			result = append(result, NetworkInterface{
				Name:      iface.Name,
				IP:        ipNet.IP.String(),
				Broadcast: broadcast.String(),
				Network:   ipNet.String(),
			})
		}
	}

	if len(result) == 0 {
		return nil, ErrNoInterfaces
	}

	return result, nil
}

// calculateBroadcast calculates the broadcast address for a given network.
func calculateBroadcast(ipNet *net.IPNet) net.IP {
	ip := ipNet.IP.To4()
	mask := ipNet.Mask

	// Calculate broadcast address: IP | ^mask
	broadcast := make(net.IP, 4)
	for i := 0; i < 4; i++ {
		broadcast[i] = ip[i] | ^mask[i]
	}

	return broadcast
}

// CreateMagicPacket creates a Wake-on-LAN magic packet for the given MAC address.
func CreateMagicPacket(macAddr string) ([]byte, error) {
	// Parse MAC address
	mac, err := parseMAC(macAddr)
	if err != nil {
		return nil, err
	}

	// Create magic packet: 6 bytes of 0xFF followed by 16 repetitions of the MAC address
	packet := make([]byte, MagicPacketSize)

	// First 6 bytes: 0xFF
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}

	// Next 96 bytes: 16 repetitions of MAC address
	for i := 0; i < 16; i++ {
		copy(packet[6+i*6:], mac)
	}

	return packet, nil
}

// parseMAC parses a MAC address in various formats.
func parseMAC(macAddr string) ([]byte, error) {
	// Remove common separators
	macAddr = strings.ReplaceAll(macAddr, ":", "")
	macAddr = strings.ReplaceAll(macAddr, "-", "")
	macAddr = strings.ReplaceAll(macAddr, ".", "")
	macAddr = strings.ToLower(macAddr)

	// MAC address should be 12 hex characters
	if len(macAddr) != 12 {
		return nil, ErrInvalidMAC
	}

	// Decode hex string
	mac, err := hex.DecodeString(macAddr)
	if err != nil {
		return nil, ErrInvalidMAC
	}

	return mac, nil
}

// SendWOL sends a Wake-on-LAN packet to the specified MAC address.
func (c *Client) SendWOL(macAddr string, broadcast string, port int) error {
	// Use default port if not specified
	if port == 0 {
		port = c.config.DefaultPort
	}

	// Create magic packet
	packet, err := CreateMagicPacket(macAddr)
	if err != nil {
		return err
	}

	// Create UDP connection
	conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", broadcast, port))
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}
	defer conn.Close()

	// Send packet
	_, err = conn.Write(packet)
	if err != nil {
		return fmt.Errorf("failed to send WOL packet: %w", err)
	}

	return nil
}

// SendWOLToAll sends a Wake-on-LAN packet to all network interfaces.
func (c *Client) SendWOLToAll(macAddr string, port int) error {
	interfaces, err := c.GetNetworkInterfaces()
	if err != nil {
		return err
	}

	var lastErr error
	successCount := 0

	for _, iface := range interfaces {
		err := c.SendWOL(macAddr, iface.Broadcast, port)
		if err != nil {
			lastErr = err
		} else {
			successCount++
		}
	}

	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to send WOL packet on any interface: %w", lastErr)
	}

	return nil
}

// GetConfig returns the current configuration.
func (c *Client) GetConfig() *Config {
	return c.config
}

// SetConfig updates the configuration.
func (c *Client) SetConfig(config *Config) {
	if config.DefaultPort == 0 {
		config.DefaultPort = DefaultPort
	}
	c.config = config
}
