package localzone

import (
	"regexp"
	"strings"
)

// TestableZoneDetector is a testable version of ZoneDetector.
type TestableZoneDetector struct {
	fileReader FileReader
	config     DetectorConfig
}

// DetectorConfig holds configuration for zone detection.
type DetectorConfig struct {
	UCIPath     string
	ResolvPath  string
	ManualZones []string

	// Detection strategies
	DetectFromUCI     bool
	DetectFromResolv  bool
	DetectFromSystemd bool
	DetectFromMDNS    bool
}

// NewTestableZoneDetector creates a testable zone detector.
func NewTestableZoneDetector(fileReader FileReader, config DetectorConfig) *TestableZoneDetector {
	return &TestableZoneDetector{
		fileReader: fileReader,
		config:     config,
	}
}

// DetectZones detects zones using the configured file reader.
func (zd *TestableZoneDetector) DetectZones() ([]string, error) { //nolint:cyclop
	zones := make([]string, 0)

	// Try UCI config detection (OpenWrt)
	if zd.config.DetectFromUCI {
		uciZones, err := zd.detectFromUCI()
		if err != nil {
			// Log warning but continue
		} else if len(uciZones) > 0 {
			zones = append(zones, uciZones...)
		}
	}

	// Try resolv.conf detection (Linux/OpenWrt)
	if zd.config.DetectFromResolv {
		resolvZones, err := zd.detectFromResolvConf()
		if err != nil {
			// Log warning but continue
		} else if len(resolvZones) > 0 {
			zones = append(zones, resolvZones...)
		}
	}

	// Try systemd-resolved detection (Linux)
	if zd.config.DetectFromSystemd && len(zones) == 0 {
		systemdZones, err := zd.detectFromSystemd()
		if err == nil && len(systemdZones) > 0 {
			zones = append(zones, systemdZones...)
		}
	}

	// Try mDNS detection (macOS/Linux)
	if zd.config.DetectFromMDNS && len(zones) == 0 {
		mdnsZones, err := zd.detectFromMDNS()
		if err == nil && len(mdnsZones) > 0 {
			zones = append(zones, mdnsZones...)
		}
	}

	// If no zones detected from files, use manual zones from config
	if len(zones) == 0 && len(zd.config.ManualZones) > 0 {
		zones = append(zones, zd.config.ManualZones...)
	}

	// If still no zones, return empty array (no fallback)
	// This allows the system to work without any local zones

	// Remove duplicates and return
	result := zd.removeDuplicates(zones)
	if result == nil {
		result = []string{}
	}

	return result, nil
}

// detectFromUCI detects zones from UCI config using file reader.
func (zd *TestableZoneDetector) detectFromUCI() ([]string, error) {
	if !zd.fileReader.FileExists(zd.config.UCIPath) {
		return nil, ErrUCIConfigNotFound
	}

	content, err := zd.fileReader.ReadFile(zd.config.UCIPath)
	if err != nil {
		return nil, err
	}

	var zones []string

	lines := strings.Split(string(content), "\n")
	domainRegex := regexp.MustCompile(`^\s*option\s+(domain|local)\s+['"]?([^'"]+)['"]?`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Look for dnsmasq sections
		if strings.Contains(line, "config dnsmasq") {
			// Parse the section
			sectionZones := zd.parseDnsmasqSectionFromLines(lines, domainRegex)
			zones = append(zones, sectionZones...)
		}
	}

	return zones, nil
}

// detectFromResolvConf detects zones from resolv.conf using file reader.
func (zd *TestableZoneDetector) detectFromResolvConf() ([]string, error) {
	if !zd.fileReader.FileExists(zd.config.ResolvPath) {
		return nil, ErrResolvConfNotFound
	}

	content, err := zd.fileReader.ReadFile(zd.config.ResolvPath)
	if err != nil {
		return nil, err
	}

	return zd.parseResolvConfContent(string(content)), nil
}

// detectFromSystemd detects zones from systemd-resolved.
func (zd *TestableZoneDetector) detectFromSystemd() ([]string, error) {
	paths := []string{
		"/run/systemd/resolve/resolv.conf",
		"/etc/systemd/resolved.conf",
		"/etc/resolv.conf",
	}

	for _, path := range paths {
		if zd.fileReader.FileExists(path) {
			content, err := zd.fileReader.ReadFile(path)
			if err != nil {
				continue
			}

			zones := zd.parseResolvConfContent(string(content))
			if len(zones) > 0 {
				return zones, nil
			}
		}
	}

	return nil, ErrSystemdConfigNotFound
}

// detectFromMDNS detects zones from mDNS.
func (zd *TestableZoneDetector) detectFromMDNS() ([]string, error) {
	paths := []string{
		"/etc/mdns.conf",
		"/etc/avahi/avahi-daemon.conf",
	}

	for _, path := range paths {
		if zd.fileReader.FileExists(path) {
			content, err := zd.fileReader.ReadFile(path)
			if err != nil {
				continue
			}

			return zd.parseMDNSContent(string(content)), nil
		}
	}

	return nil, ErrMDNSConfigNotFound
}

// parseResolvConfContent parses resolv.conf content.
func (zd *TestableZoneDetector) parseResolvConfContent(content string) []string {
	var zones []string

	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

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

	return zones
}

// parseMDNSContent parses mDNS configuration content.
func (zd *TestableZoneDetector) parseMDNSContent(content string) []string {
	var zones []string

	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for domain-name in avahi config
		if strings.HasPrefix(line, "domain-name=") {
			parts := strings.Split(line, "=")
			if len(parts) > 1 {
				zones = append(zones, parts[1])
			}
		}
	}

	return zones
}

// parseDnsmasqSectionFromLines parses dnsmasq section from lines.
func (zd *TestableZoneDetector) parseDnsmasqSectionFromLines(lines []string, domainRegex *regexp.Regexp) []string {
	var zones []string
	// Simple implementation - look for domain/local options in the lines
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "option domain") {
			matches := domainRegex.FindStringSubmatch(line)

			const minMatches = 3
			if len(matches) > minMatches-1 {
				zones = append(zones, matches[2])
			}
		} else if strings.HasPrefix(line, "option local") {
			matches := domainRegex.FindStringSubmatch(line)

			const minMatches = 3
			if len(matches) > minMatches-1 {
				// Extract domain from /domain/ format
				domain := strings.Trim(matches[2], "/")
				zones = append(zones, domain)
			}
		}
	}

	return zones
}

// removeDuplicates removes duplicate zones.
func (zd *TestableZoneDetector) removeDuplicates(zones []string) []string {
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
