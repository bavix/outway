package wol

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"

	customerrors "github.com/bavix/outway/internal/errors"
)

// WakeOnLan represents a Wake-on-LAN packet sender.
type WakeOnLan struct {
	configManager     *ConfigManager
	interfaceDetector *InterfaceDetector
}

// NewWakeOnLan creates a new Wake-on-LAN instance.
func NewWakeOnLan() *WakeOnLan {
	return &WakeOnLan{
		configManager:     NewConfigManager(),
		interfaceDetector: NewInterfaceDetector(),
	}
}

// NewWakeOnLanWithConfig creates a new Wake-on-LAN instance with custom configuration.
func NewWakeOnLanWithConfig(config *Config) *WakeOnLan {
	wol := &WakeOnLan{
		configManager:     NewConfigManager(),
		interfaceDetector: NewInterfaceDetector(),
	}

	if config != nil {
		_ = wol.configManager.SetConfig(config)
	}

	return wol
}

// MagicPacket represents a Wake-on-LAN magic packet.
type MagicPacket struct {
	MAC     string `json:"mac"`
	IP      string `json:"ip,omitempty"`
	Port    int    `json:"port,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

// WakeOnLanRequest represents a Wake-on-LAN request.
type WakeOnLanRequest struct {
	MAC     string `json:"mac"               validate:"required"`
	IP      string `json:"ip,omitempty"`
	Port    int    `json:"port,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

// WakeOnLanResponse represents a Wake-on-LAN response.
type WakeOnLanResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	MAC     string `json:"mac"`
	IP      string `json:"ip,omitempty"`
	Port    int    `json:"port,omitempty"`
}

// SendMagicPacket sends a Wake-on-LAN magic packet.
func (w *WakeOnLan) SendMagicPacket(ctx context.Context, req *WakeOnLanRequest) (*WakeOnLanResponse, error) {
	// Validate MAC address
	mac, err := w.normalizeMAC(req.MAC)
	if err != nil {
		return &WakeOnLanResponse{
			Success: false,
			Message: fmt.Sprintf("invalid MAC address: %v", err),
			MAC:     req.MAC,
		}, err
	}

	// Set defaults
	ip := req.IP
	if ip == "" {
		ip = "255.255.255.255" // Broadcast
	}

	port := req.Port
	if port == 0 {
		port = 9 // Default WOL port
	}

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 5 // Default timeout in seconds
	}

	// Create magic packet
	packet, err := w.createMagicPacket(mac)
	if err != nil {
		return &WakeOnLanResponse{
			Success: false,
			Message: fmt.Sprintf("failed to create magic packet: %v", err),
			MAC:     mac,
			IP:      ip,
			Port:    port,
		}, err
	}

	// Send packet
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	err = w.sendPacket(ctx, packet, ip, port)
	if err != nil {
		return &WakeOnLanResponse{
			Success: false,
			Message: fmt.Sprintf("failed to send packet: %v", err),
			MAC:     mac,
			IP:      ip,
			Port:    port,
		}, err
	}

	zerolog.Ctx(ctx).Info().
		Str("mac", mac).
		Str("ip", ip).
		Int("port", port).
		Msg("Wake-on-LAN packet sent successfully")

	return &WakeOnLanResponse{
		Success: true,
		Message: "Wake-on-LAN packet sent successfully",
		MAC:     mac,
		IP:      ip,
		Port:    port,
	}, nil
}

// ValidateMAC validates a MAC address format.
func (w *WakeOnLan) ValidateMAC(mac string) error {
	_, err := w.normalizeMAC(mac)

	return err
}

// GetBroadcastAddresses returns all available broadcast addresses.
func (w *WakeOnLan) GetBroadcastAddresses(ctx context.Context) ([]string, error) {
	return w.interfaceDetector.GetBroadcastAddresses(ctx)
}

// GetInterfaces returns all available network interfaces.
func (w *WakeOnLan) GetInterfaces(ctx context.Context) ([]NetworkInterface, error) {
	return w.interfaceDetector.DetectInterfaces(ctx)
}

// GetBestInterface returns the best interface for Wake-on-LAN.
func (w *WakeOnLan) GetBestInterface(ctx context.Context) (*NetworkInterface, error) {
	return w.interfaceDetector.GetBestInterface(ctx)
}

// GetConfig returns the current configuration.
func (w *WakeOnLan) GetConfig() *Config {
	return w.configManager.GetConfig()
}

// SetConfig updates the configuration.
func (w *WakeOnLan) SetConfig(config *Config) error {
	return w.configManager.SetConfig(config)
}

// UpdateConfig updates specific configuration fields.
func (w *WakeOnLan) UpdateConfig(updates map[string]any) error {
	return w.configManager.UpdateConfig(updates)
}

// IsEnabled returns whether Wake-on-LAN is enabled.
func (w *WakeOnLan) IsEnabled() bool {
	return w.configManager.IsEnabled()
}

// SendMagicPacketToInterface sends a Wake-on-LAN packet to a specific interface.
func (w *WakeOnLan) SendMagicPacketToInterface(
	ctx context.Context,
	req *WakeOnLanRequest,
	interfaceName string,
) (*WakeOnLanResponse, error) {
	// Get the specific interface
	iface, err := w.interfaceDetector.GetInterfaceByName(ctx, interfaceName)
	if err != nil {
		return &WakeOnLanResponse{
			Success: false,
			Message: fmt.Sprintf("interface not found: %v", err),
			MAC:     req.MAC,
		}, err
	}

	// Validate interface
	if err := w.interfaceDetector.ValidateInterface(iface); err != nil {
		return &WakeOnLanResponse{
			Success: false,
			Message: fmt.Sprintf("invalid interface: %v", err),
			MAC:     req.MAC,
		}, err
	}

	// Use interface's broadcast address if no IP specified
	if req.IP == "" {
		req.IP = iface.Broadcast
	}

	return w.SendMagicPacket(ctx, req)
}

// SendMagicPacketToAllInterfaces sends a Wake-on-LAN packet to all suitable interfaces.
func (w *WakeOnLan) SendMagicPacketToAllInterfaces(ctx context.Context, req *WakeOnLanRequest) ([]WakeOnLanResponse, error) {
	interfaces, err := w.GetInterfaces(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	responses := make([]WakeOnLanResponse, 0, len(interfaces))

	var lastErr error

	for _, iface := range interfaces {
		// Skip invalid interfaces
		if err := w.interfaceDetector.ValidateInterface(&iface); err != nil {
			zerolog.Ctx(ctx).Warn().
				Str("interface", iface.Name).
				Err(err).
				Msg("skipping invalid interface")

			continue
		}

		// Create request for this interface
		ifaceReq := *req
		if ifaceReq.IP == "" {
			ifaceReq.IP = iface.Broadcast
		}

		// Send packet
		resp, err := w.SendMagicPacket(ctx, &ifaceReq)
		if err != nil {
			zerolog.Ctx(ctx).Warn().
				Str("interface", iface.Name).
				Err(err).
				Msg("failed to send packet to interface")
			lastErr = err
		}

		responses = append(responses, *resp)
	}

	if len(responses) == 0 {
		return nil, fmt.Errorf("no valid interfaces found: %w", lastErr)
	}

	return responses, nil
}

// SendMagicPacketWithRetry sends a Wake-on-LAN packet with retry logic.
func (w *WakeOnLan) SendMagicPacketWithRetry(ctx context.Context, req *WakeOnLanRequest) (*WakeOnLanResponse, error) {
	config := w.GetConfig()

	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		resp, err := w.SendMagicPacket(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err
		zerolog.Ctx(ctx).Warn().
			Int("attempt", attempt+1).
			Int("max_retries", config.MaxRetries).
			Err(err).
			Msg("Wake-on-LAN attempt failed")

		// Don't wait after the last attempt
		if attempt < config.MaxRetries {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(config.RetryDelay):
				// Continue to next attempt
			}
		}
	}

	return &WakeOnLanResponse{
		Success: false,
		Message: fmt.Sprintf("failed after %d attempts: %v", config.MaxRetries+1, lastErr),
		MAC:     req.MAC,
		IP:      req.IP,
		Port:    req.Port,
	}, lastErr
}

// normalizeMAC normalizes a MAC address to the standard format.
func (w *WakeOnLan) normalizeMAC(mac string) (string, error) {
	// Remove common separators
	mac = strings.ReplaceAll(mac, ":", "")
	mac = strings.ReplaceAll(mac, "-", "")
	mac = strings.ReplaceAll(mac, ".", "")
	mac = strings.ReplaceAll(mac, " ", "")

	// Convert to lowercase
	mac = strings.ToLower(mac)

	// Validate length (12 hex characters)
	if len(mac) != macAddressLength {
		return "", customerrors.ErrMACAddressInvalidLengthWithLength(len(mac))
	}

	// Validate hex characters
	for _, c := range mac {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return "", customerrors.ErrMACAddressInvalidCharacters
		}
	}

	// Format as XX:XX:XX:XX:XX:XX
	formatted := fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		mac[0:2], mac[2:4], mac[4:6],
		mac[6:8], mac[8:10], mac[10:12])

	return formatted, nil
}

// createMagicPacket creates a Wake-on-LAN magic packet.
func (w *WakeOnLan) createMagicPacket(mac string) ([]byte, error) {
	// Remove separators for parsing
	macBytes := strings.ReplaceAll(mac, ":", "")
	macBytes = strings.ReplaceAll(macBytes, "-", "")

	// Parse MAC address
	macAddr, err := hex.DecodeString(macBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to decode MAC address: %w", err)
	}

	if len(macAddr) != macBytesLength {
		return nil, customerrors.ErrMACAddressInvalidBytesWithLength(len(macAddr))
	}

	// Create magic packet: 6 bytes of 0xFF + 16 repetitions of MAC address
	packet := make([]byte, magicPacketSize) // 6 + 16*6 = 102 bytes

	// Fill first 6 bytes with 0xFF
	for i := range 6 {
		packet[i] = 0xFF
	}

	// Repeat MAC address 16 times
	for i := range 16 {
		copy(packet[6+i*6:6+(i+1)*6], macAddr)
	}

	return packet, nil
}

// sendPacket sends the magic packet via UDP.
func (w *WakeOnLan) sendPacket(ctx context.Context, packet []byte, ip string, port int) error {
	// Resolve UDP address
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		return fmt.Errorf("failed to resolve address: %w", err)
	}

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("failed to create UDP connection: %w", err)
	}

	defer func() { _ = conn.Close() }()

	// Set write deadline
	deadline, ok := ctx.Deadline()
	if ok {
		if err := conn.SetWriteDeadline(deadline); err != nil {
			return fmt.Errorf("failed to set write deadline: %w", err)
		}
	}

	// Send packet
	_, err = conn.Write(packet)
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	return nil
}
