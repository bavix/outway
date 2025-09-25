package dnsproxy

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

const (
	dotPort = 853
)

// DetectTypeFromAddress infers upstream type from address/scheme/port.
// Returns one of: doh, doq, dot, tcp, udp.
func DetectTypeFromAddress(address string) string { //nolint:cyclop
	a := strings.TrimSpace(address)
	if a == "" {
		return protocolUDP
	}
	// Scheme-based
	if u, err := url.Parse(a); err == nil && u.Scheme != "" {
		switch strings.ToLower(u.Scheme) {
		case "https":
			return protocolDOH
		case "quic":
			return protocolDOQ
		case "tls":
			return protocolDOT
		case protocolTCP:
			return protocolTCP
		case protocolUDP:
			return protocolUDP
		}
	}
	// Prefix proto:host:port legacy
	if strings.HasPrefix(a, "udp:") {
		return protocolUDP
	}

	if strings.HasPrefix(a, "tcp:") {
		return protocolTCP
	}

	if strings.HasPrefix(a, "doh:") {
		return protocolDOH
	}

	if strings.HasPrefix(a, "dot:") || strings.HasPrefix(a, "tls:") {
		return protocolDOT
	}

	if strings.HasPrefix(a, "doq:") || strings.HasPrefix(a, "quic:") {
		return protocolDOQ
	}

	// Port-based: 853 -> DoT by default
	host, port, err := net.SplitHostPort(strings.TrimPrefix(a, "tls:"))
	if err == nil {
		if p, _ := strconv.Atoi(port); p == dotPort {
			_ = host

			return protocolDOT
		}

		return protocolUDP // explicit host:port default to udp
	}
	// No port -> udp:53 assumed
	return protocolUDP
}
