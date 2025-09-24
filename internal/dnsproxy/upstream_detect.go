package dnsproxy

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

// DetectTypeFromAddress infers upstream type from address/scheme/port.
// Returns one of: doh, doq, dot, tcp, udp
func DetectTypeFromAddress(address string) string {
	a := strings.TrimSpace(address)
	if a == "" {
		return "udp"
	}
	// Scheme-based
	if u, err := url.Parse(a); err == nil && u.Scheme != "" {
		switch strings.ToLower(u.Scheme) {
		case "https":
			return "doh"
		case "quic":
			return "doq"
		case "tls":
			return "dot"
		case "tcp":
			return "tcp"
		case "udp":
			return "udp"
		}
	}
	// Prefix proto:host:port legacy
	if strings.HasPrefix(a, "udp:") {
		return "udp"
	}
	if strings.HasPrefix(a, "tcp:") {
		return "tcp"
	}
	if strings.HasPrefix(a, "doh:") {
		return "doh"
	}
	if strings.HasPrefix(a, "dot:") || strings.HasPrefix(a, "tls:") {
		return "dot"
	}
	if strings.HasPrefix(a, "doq:") || strings.HasPrefix(a, "quic:") {
		return "doq"
	}

	// Port-based: 853 -> DoT by default
	host, port, err := net.SplitHostPort(strings.TrimPrefix(a, "tls:"))
	if err == nil {
		if p, _ := strconv.Atoi(port); p == 853 {
			_ = host
			return "dot"
		}
		return "udp" // explicit host:port default to udp
	}
	// No port -> udp:53 assumed
	return "udp"
}
