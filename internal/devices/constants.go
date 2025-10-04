package devices

// Device status constants.
const (
	StatusOnline  = "online"
	StatusOffline = "offline"
	StatusUnknown = "unknown"
)

// Device source constants.
const (
	SourceDHCPLeases  = "dhcp"
	SourceNetworkScan = "scan"
	SourceARP         = "arp"
	SourceManual      = "manual"
	SourceMultiple    = "multiple"
)

// Device type constants.
const (
	DeviceTypeComputer = "computer"
	DeviceTypePhone    = "phone"
	DeviceTypeTablet   = "tablet"
	DeviceTypeRouter   = "router"
	DeviceTypeTV       = "tv"
	DeviceTypeOther    = "other"
)

// Priority constants.
const (
	PriorityDHCPLeases  = 100 // Highest priority - DHCP is the source of truth
	PriorityNetworkScan = 50  // Lower priority than DHCP
)

// ID generation constants.
const (
	RandomIDLength = 6 // Length of random string in device ID
)
