package firewall

import (
	"errors"
	"net"
	"regexp"
)

var (
	ErrInvalidIface = errors.New("invalid interface name")
	ErrInvalidIP    = errors.New("invalid IP address")
)

const (
	minTTLSeconds = 30 // Minimum TTL in seconds
)

// Interface name validation regex.
var IfaceNameRe = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,32}$`)

// IsSafeIfaceName verifies interface names to a conservative charset to avoid injection via args.
func IsSafeIfaceName(iface string) bool {
	return IfaceNameRe.MatchString(iface)
}

// NormalizeIP parses and returns canonical string representation without brackets.
func NormalizeIP(raw string) (string, bool) {
	ip := net.ParseIP(raw)
	if ip == nil {
		return "", false
	}

	// Return IPv4 in dotted decimal, IPv6 in canonical form
	if ip.To4() != nil {
		return ip.To4().String(), true
	}

	return ip.String(), true
}

// PFTableName generates a table name for pf backend.
func PFTableName(iface string) string {
	return "outway_" + iface
}
