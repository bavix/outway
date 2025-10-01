//nolint:funcorder
package localzone

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
)

var (
	// ErrUCIConfigNotFound is returned when UCI config file is not found.
	ErrUCIConfigNotFound = errors.New("UCI config file not found")
	// ErrResolvConfNotFound is returned when resolv.conf file is not found.
	ErrResolvConfNotFound = errors.New("resolv.conf file not found")
	// ErrSystemdConfigNotFound is returned when systemd-resolved config is not found.
	ErrSystemdConfigNotFound = errors.New("systemd-resolved config not found")
	// ErrMDNSConfigNotFound is returned when mDNS config is not found.
	ErrMDNSConfigNotFound = errors.New("mDNS config not found")
)

// ZoneDetector detects local zones from various sources.
type ZoneDetector struct {
	// Paths to check for zone configuration
	UCIConfigPath  string
	ResolvConfPath string
	ManualZones    []string

	// Detection strategies
	DetectFromUCI     bool
	DetectFromResolv  bool
	DetectFromSystemd bool
	DetectFromMDNS    bool
}

// NewZoneDetector creates a new zone detector with full auto-detection.
func NewZoneDetector() *ZoneDetector {
	return &ZoneDetector{
		UCIConfigPath:  "/etc/config/dhcp",      // Default OpenWrt path
		ResolvConfPath: "/tmp/resolv.conf.auto", // Default OpenWrt path
		ManualZones:    []string{},              // No manual zones - auto-detect everything

		// All detection strategies enabled
		DetectFromUCI:     true, // Try UCI config
		DetectFromResolv:  true, // Try resolv.conf
		DetectFromSystemd: true, // Try systemd-resolved
		DetectFromMDNS:    true, // Try mDNS
	}
}

// NewZoneDetectorWithConfig creates a new zone detector with full configuration.
func NewZoneDetectorWithConfig(manualZones []string, uciPath, resolvPath string, detectUCI, detectResolv, detectSystemd, detectMDNS bool) *ZoneDetector { //nolint:lll
	// Auto-detect paths if not provided
	if uciPath == "" {
		uciPath = "/etc/config/dhcp" // Default OpenWrt path
	}

	if resolvPath == "" {
		resolvPath = "/tmp/resolv.conf.auto" // Default OpenWrt path
	}

	return &ZoneDetector{
		UCIConfigPath:  uciPath,
		ResolvConfPath: resolvPath,
		ManualZones:    manualZones,

		// Use provided detection strategies
		DetectFromUCI:     detectUCI,
		DetectFromResolv:  detectResolv,
		DetectFromSystemd: detectSystemd,
		DetectFromMDNS:    detectMDNS,
	}
}

// DetectZones detects all local zones from available sources.
func (zd *ZoneDetector) DetectZones() ([]string, error) { //nolint:cyclop
	zones := make([]string, 0)

	// Try UCI config detection (OpenWrt)
	if zd.DetectFromUCI {
		uciZones, err := zd.detectFromUCI()
		if err != nil {
			// Continue with other detection methods
		} else if len(uciZones) > 0 {
			zones = append(zones, uciZones...)
		}
	}

	// Try resolv.conf detection (Linux/OpenWrt)
	if zd.DetectFromResolv {
		resolvZones, err := zd.detectFromResolvConf()
		if err != nil {
			// Continue with other detection methods
		} else if len(resolvZones) > 0 {
			zones = append(zones, resolvZones...)
		}
	}

	// Try systemd-resolved detection (Linux)
	if zd.DetectFromSystemd && len(zones) == 0 {
		systemdZones, err := zd.detectFromSystemd()
		if err != nil {
			// Continue with other detection methods
		} else if len(systemdZones) > 0 {
			zones = append(zones, systemdZones...)
		}
	}

	// Try mDNS detection (macOS/Linux)
	if zd.DetectFromMDNS && len(zones) == 0 {
		mdnsZones, err := zd.detectFromMDNS()
		if err != nil {
			// Continue with other detection methods
		} else if len(mdnsZones) > 0 {
			zones = append(zones, mdnsZones...)
		}
	}

	// If no zones detected from files, use manual zones from config (if provided)
	if len(zones) == 0 && len(zd.ManualZones) > 0 {
		zones = append(zones, zd.ManualZones...)
	}

	// If still no zones, try common fallback domains
	if len(zones) == 0 {
		commonDomains := []string{"local", "lan", "home", "internal"}
		zones = append(zones, commonDomains...)
	}

	// Remove duplicates and return
	result := zd.removeDuplicates(zones)
	if result == nil {
		result = []string{}
	}

	return result, nil
}

// detectFromUCI detects zones from OpenWrt UCI config.
//
//nolint:funcorder
func (zd *ZoneDetector) detectFromUCI() ([]string, error) {
	if _, err := os.Stat(zd.UCIConfigPath); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrUCIConfigNotFound, zd.UCIConfigPath)
	}

	file, err := os.Open(zd.UCIConfigPath)
	if err != nil {
		return nil, err
	}

	defer func() { _ = file.Close() }()

	var zones []string

	scanner := bufio.NewScanner(file)

	// Regex to match domain and local options in dnsmasq sections
	domainRegex := regexp.MustCompile(`^\s*option\s+(domain|local)\s+['"]?([^'"]+)['"]?`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Look for dnsmasq sections
		if strings.Contains(line, "config dnsmasq") {
			// Read the section
			sectionZones := zd.parseDnsmasqSection(scanner, domainRegex)
			zones = append(zones, sectionZones...)
		}
	}

	return zones, scanner.Err()
}

// detectFromMDNS detects zones from mDNS (macOS/Linux).
func (zd *ZoneDetector) detectFromMDNS() ([]string, error) {
	// Try to read mDNS configuration
	// /etc/mdns.conf or /etc/avahi/avahi-daemon.conf
	paths := []string{
		"/etc/mdns.conf",
		"/etc/avahi/avahi-daemon.conf",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return zd.detectFromMDNSFile(path)
		}
	}

	// Fallback: try to detect common local domains
	commonDomains := []string{"local", "lan", "home", "internal"}

	return commonDomains, nil
}

// detectFromResolvConfFile detects zones from a specific resolv.conf file.
func (zd *ZoneDetector) detectFromResolvConfFile(path string) ([]string, error) {
	file, err := os.Open(path) //nolint:gosec
	if err != nil {
		return nil, err
	}

	defer func() { _ = file.Close() }()

	var zones []string

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for search/domain lines
		if strings.HasPrefix(line, "search ") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				zones = append(zones, parts[1:]...)
			}
		} else if strings.HasPrefix(line, "domain ") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				zones = append(zones, parts[1])
			}
		}
	}

	return zones, scanner.Err()
}

// detectFromMDNSFile detects zones from mDNS configuration file.
func (zd *ZoneDetector) detectFromMDNSFile(path string) ([]string, error) {
	file, err := os.Open(path) //nolint:gosec
	if err != nil {
		return nil, err
	}

	defer func() { _ = file.Close() }()

	var zones []string

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for domain patterns in mDNS config
		if strings.Contains(line, "domain") || strings.Contains(line, "local") {
			// Simple pattern matching for common mDNS configs
			if strings.Contains(line, ".local") {
				zones = append(zones, "local")
			}

			if strings.Contains(line, ".lan") {
				zones = append(zones, "lan")
			}

			if strings.Contains(line, ".home") {
				zones = append(zones, "home")
			}
		}
	}

	return zones, scanner.Err()
}

// parseDnsmasqSection parses a dnsmasq section for domain/local options.
//
//nolint:funcorder
func (zd *ZoneDetector) parseDnsmasqSection(scanner *bufio.Scanner, domainRegex *regexp.Regexp) []string {
	var zones []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// End of section
		if strings.HasPrefix(line, "config ") {
			break
		}

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Check for domain or local options
		matches := domainRegex.FindStringSubmatch(line)

		const minMatches = 3
		if len(matches) >= minMatches {
			zone := matches[2]
			// Remove leading/trailing slashes and quotes
			zone = strings.Trim(zone, "/\"'")
			if zone != "" {
				zones = append(zones, zone)
			}
		}
	}

	return zones
}

// detectFromResolvConf detects zones from resolv.conf.auto.
//
//nolint:funcorder
func (zd *ZoneDetector) detectFromResolvConf() ([]string, error) {
	if _, err := os.Stat(zd.ResolvConfPath); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrResolvConfNotFound, zd.ResolvConfPath)
	}

	file, err := os.Open(zd.ResolvConfPath)
	if err != nil {
		return nil, err
	}

	defer func() { _ = file.Close() }()

	var zones []string

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Look for search or domain lines
		if strings.HasPrefix(line, "search ") || strings.HasPrefix(line, "domain ") {
			parts := strings.Fields(line)
			if len(parts) > 1 {
				// Extract domain from search/domain line
				domain := parts[1]
				// Remove trailing dot if present
				domain = strings.TrimSuffix(domain, ".")
				if domain != "" {
					zones = append(zones, domain)
				}
			}
		}
	}

	return zones, scanner.Err()
}

// removeDuplicates removes duplicate zones from the slice.
//
//nolint:funcorder
func (zd *ZoneDetector) removeDuplicates(zones []string) []string {
	seen := make(map[string]bool)

	result := make([]string, 0, len(zones))

	for _, zone := range zones {
		if !seen[zone] {
			seen[zone] = true
			result = append(result, zone)
		}
	}

	return result
}

// IsLocalZone checks if a domain belongs to any of the detected local zones.
func (zd *ZoneDetector) IsLocalZone(domain string) (bool, string) {
	zones, _ := zd.DetectZones()

	domain = strings.ToLower(domain)
	domain = strings.TrimSuffix(domain, ".")

	for _, zone := range zones {
		zone = strings.ToLower(zone)
		if strings.HasSuffix(domain, "."+zone) || domain == zone {
			return true, zone
		}
	}

	return false, ""
}

// detectFromSystemd detects zones from systemd-resolved (Linux).
func (zd *ZoneDetector) detectFromSystemd() ([]string, error) {
	// Try to read systemd-resolved configuration
	// /etc/systemd/resolved.conf or /run/systemd/resolve/resolv.conf
	paths := []string{
		"/run/systemd/resolve/resolv.conf",
		"/etc/systemd/resolved.conf",
		"/etc/resolv.conf",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return zd.detectFromResolvConfFile(path)
		}
	}

	return nil, ErrSystemdConfigNotFound
}
