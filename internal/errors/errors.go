package errors

import (
	"errors"
)

// Common errors.
var (
	ErrConfigCannotBeNil            = errors.New("config cannot be nil")
	ErrNetworkScannerNotInitialized = errors.New("network scanner not initialized")
	ErrDeviceNotFound               = errors.New("device not found")
	ErrDeviceCannotBeWoken          = errors.New("device cannot be woken up")
	ErrDeviceHostnameNotFound       = errors.New("device with hostname not found")
	ErrDeviceMACAlreadyExists       = errors.New("device with MAC already exists")
	ErrFileWatcherAlreadyEnabled    = errors.New("file watcher already enabled")
	ErrFileWatcherNotImplemented    = errors.New("file watcher not implemented")
	ErrNoSuitableNetworkInterfaces  = errors.New("no suitable network interfaces found")
	ErrInterfaceNotFound            = errors.New("interface not found")
	ErrInterfaceIsNil               = errors.New("interface is nil")
	ErrLoopbackInterfaceNotSuitable = errors.New("loopback interface not suitable for Wake-on-LAN")
	ErrInterfaceIsDown              = errors.New("interface is down")
	ErrInterfaceNoBroadcastAddress  = errors.New("interface has no broadcast address")
	ErrUnsupportedOSForARPTable     = errors.New("unsupported OS for ARP table")
	ErrNoLocalNetworkFound          = errors.New("no local network found")
	ErrMACAddressInvalidLength      = errors.New("MAC address must be 12 hex characters")
	ErrMACAddressInvalidCharacters  = errors.New("MAC address contains invalid characters")
	ErrMACAddressInvalidBytes       = errors.New("MAC address must be 6 bytes")
	ErrRequiredToolNotFound         = errors.New("required tool not found")
	ErrMergerNotSet                 = errors.New("device merger not set")
	ErrLeaseManagerNotInitialized   = errors.New("lease manager not initialized")
	ErrSOLNotSupported              = errors.New("device does not support SoL")
	ErrSOLPacketSendFailed          = errors.New("failed to send SoL packet")
)

// ErrDeviceNotFoundWithID returns an error for device not found with ID.
func ErrDeviceNotFoundWithID(deviceID string) error {
	return ErrDeviceNotFound
}

func ErrDeviceCannotBeWokenUp(deviceName string) error {
	return ErrDeviceCannotBeWoken
}

func ErrDeviceNotFoundWithHostname(hostname string) error {
	return ErrDeviceHostnameNotFound
}

func ErrDeviceAlreadyExistsWithMAC(mac string) error {
	return ErrDeviceMACAlreadyExists
}

func ErrInvalidTypeForDefaultPort(value any) error {
	return ErrConfigCannotBeNil
}

func ErrInvalidTypeForDefaultTimeout(value any) error {
	return ErrConfigCannotBeNil
}

func ErrInvalidTypeForMaxRetries(value any) error {
	return ErrConfigCannotBeNil
}

func ErrInvalidTypeForRetryDelay(value any) error {
	return ErrConfigCannotBeNil
}

func ErrInvalidTypeForEnabled(value any) error {
	return ErrConfigCannotBeNil
}

func ErrUnknownConfigurationKey(key string) error {
	return ErrConfigCannotBeNil
}

func ErrDefaultPortInvalid(port int) error {
	return ErrConfigCannotBeNil
}

func ErrDefaultTimeoutInvalid(timeout any) error {
	return ErrConfigCannotBeNil
}

func ErrMaxRetriesInvalid(retries int) error {
	return ErrConfigCannotBeNil
}

func ErrRetryDelayInvalid(delay any) error {
	return ErrConfigCannotBeNil
}

func ErrInterfaceNotFoundWithName(name string) error {
	return ErrInterfaceNotFound
}

func ErrMACAddressInvalidLengthWithLength(length int) error {
	return ErrMACAddressInvalidLength
}

func ErrMACAddressInvalidBytesWithLength(length int) error {
	return ErrMACAddressInvalidBytes
}

func ErrRequiredToolNotFoundWithTool(tool string) error {
	return ErrRequiredToolNotFound
}
