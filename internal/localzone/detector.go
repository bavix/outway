package localzone

import (
	"bufio"
	"os"
	"strings"
)

// DetectLocalZones detects local DNS zones from OpenWrt UCI config and resolv.conf.
// Priority: 1) explicit override 2) UCI config 3) resolv.conf.auto
func DetectLocalZones(overrideZones []string, uciPath, resolvPath string) []string {
	// Use explicit override if provided
	if len(overrideZones) > 0 {
		return cleanZones(overrideZones)
	}

	// Try UCI config first
	zones := detectFromUCI(uciPath)
	if len(zones) > 0 {
		return cleanZones(zones)
	}

	// Fallback to resolv.conf.auto
	zones = detectFromResolv(resolvPath)
	return cleanZones(zones)
}

// detectFromUCI parses OpenWrt UCI dhcp config for local zones.
// Looks for "domain" and "local" options in all dnsmasq sections.
func detectFromUCI(path string) []string {
	if path == "" {
		path = "/etc/config/dhcp"
	}

	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	var zones []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for domain or local options in dnsmasq config
		// Example: option domain 'lan'
		// Example: option local '/home/'
		if strings.HasPrefix(line, "option domain") || strings.HasPrefix(line, "option local") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				zone := strings.Trim(parts[2], "'/\"")
				zone = strings.Trim(zone, "/") // Remove leading/trailing slashes
				if zone != "" {
					zones = append(zones, zone)
				}
			}
		}
	}

	return zones
}

// detectFromResolv parses resolv.conf for search/domain directives.
func detectFromResolv(path string) []string {
	if path == "" {
		path = "/tmp/resolv.conf.auto"
	}

	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = file.Close() }()

	var zones []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for "search" or "domain" lines
		if strings.HasPrefix(line, "search ") || strings.HasPrefix(line, "domain ") {
			parts := strings.Fields(line)
			for i := 1; i < len(parts); i++ {
				zone := strings.TrimSpace(parts[i])
				if zone != "" {
					zones = append(zones, zone)
				}
			}
		}
	}

	return zones
}

// cleanZones normalizes zone names (lowercase, no dots at end).
func cleanZones(zones []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, z := range zones {
		z = strings.ToLower(strings.TrimSpace(z))
		z = strings.Trim(z, ".")
		if z != "" && !seen[z] {
			seen[z] = true
			result = append(result, z)
		}
	}

	return result
}
