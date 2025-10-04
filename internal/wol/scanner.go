package wol

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	customerrors "github.com/bavix/outway/internal/errors"
)

const (
	linuxOS  = "linux"
	darwinOS = "darwin"

	// Timeout constants.
	nmapTimeout     = 30 * time.Second
	pingTimeout     = 2 * time.Second
	hostnameTimeout = 5 * time.Second

	// Network constants.
	maxConcurrentPings = 50
	macAddressParts    = 6
	macPartLength      = 2

	// ARP table parsing constants.
	arpTableMinParts   = 5
	arpScanMinParts    = 2
	arpScanVendorParts = 3
	arpMinParts        = 4
)

// NetworkScanner scans the local network for devices.
type NetworkScanner struct {
	timeout time.Duration
}

// NewNetworkScanner creates a new network scanner.
func NewNetworkScanner() *NetworkScanner {
	return &NetworkScanner{
		timeout: scanTimeout, // Уменьшили таймаут для быстрого сканирования
	}
}

// ScanResult represents a discovered device.
type ScanResult struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Hostname string `json:"hostname"`
	Vendor   string `json:"vendor"`
	Status   string `json:"status"`
}

// ScanNetwork scans the local network for devices.
func (ns *NetworkScanner) ScanNetwork(ctx context.Context, networkCIDR string) ([]*ScanResult, error) {
	// Parse network CIDR
	_, ipNet, err := net.ParseCIDR(networkCIDR)
	if err != nil {
		return nil, fmt.Errorf("invalid network CIDR: %w", err)
	}

	var results []*ScanResult

	switch runtime.GOOS {
	case linuxOS:
		results = ns.scanLinux(ctx, ipNet)
	case darwinOS:
		results = ns.scanMacOS(ctx, ipNet)
	default:
		// Fallback to basic ping scan
		results = ns.scanBasic(ctx, ipNet)
	}

	// Return empty slice instead of nil if no results found
	if results == nil {
		results = []*ScanResult{}
	}

	return results, nil
}

// GetLocalNetworkCIDR returns the local network CIDR.
func (ns *NetworkScanner) GetLocalNetworkCIDR() (string, error) {
	// Get local interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Skip loopback and inactive interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok {
				// Skip IPv6 and loopback
				if ipNet.IP.To4() == nil || ipNet.IP.IsLoopback() {
					continue
				}

				// Return the first valid IPv4 network
				return ipNet.String(), nil
			}
		}
	}

	return "", customerrors.ErrNoLocalNetworkFound
}

// scanLinux performs network scan on Linux using nmap or arp-scan.
func (ns *NetworkScanner) scanLinux(ctx context.Context, ipNet *net.IPNet) []*ScanResult {
	// Try nmap first
	if ns.hasCommand("nmap") {
		results, err := ns.scanWithNmap(ctx, ipNet)
		if err == nil {
			return results
		}
		// Fallback to basic scan if nmap fails
	}

	// Try arp-scan
	if ns.hasCommand("arp-scan") {
		results, err := ns.scanWithArpScan(ctx, ipNet)
		if err == nil {
			return results
		}
		// Fallback to basic scan if arp-scan fails
	}

	// Fallback to basic ARP scan
	results, err := ns.scanWithArp(ctx, ipNet)
	if err == nil {
		return results
	}

	// Final fallback to basic ping scan
	return ns.scanBasic(ctx, ipNet)
}

// scanMacOS performs network scan on macOS using nmap or arp-scan.
func (ns *NetworkScanner) scanMacOS(ctx context.Context, ipNet *net.IPNet) []*ScanResult {
	// Try nmap first
	if ns.hasCommand("nmap") {
		results, err := ns.scanWithNmap(ctx, ipNet)
		if err == nil {
			return results
		}
		// Fallback to basic scan if nmap fails
	}

	// Try arp-scan
	if ns.hasCommand("arp-scan") {
		results, err := ns.scanWithArpScan(ctx, ipNet)
		if err == nil {
			return results
		}
		// Fallback to basic scan if arp-scan fails
	}

	// Fallback to basic ARP scan
	results, err := ns.scanWithArp(ctx, ipNet)
	if err == nil {
		return results
	}

	// Final fallback to basic ping scan
	return ns.scanBasic(ctx, ipNet)
}

// scanWithNmap performs network scan using nmap.
func (ns *NetworkScanner) scanWithNmap(ctx context.Context, ipNet *net.IPNet) ([]*ScanResult, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, nmapTimeout)
	defer cancel()

	// Run nmap command
	cmd := exec.CommandContext(ctx, "nmap", "-sn", ipNet.String()) // #nosec G204

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nmap scan failed: %w", err)
	}

	// Parse output
	return ns.parseNmapOutput(string(output)), nil
}

// scanWithArpScan performs network scan using arp-scan.
func (ns *NetworkScanner) scanWithArpScan(ctx context.Context, ipNet *net.IPNet) ([]*ScanResult, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, nmapTimeout)
	defer cancel()

	// Run arp-scan command
	cmd := exec.CommandContext(ctx, "arp-scan", "-l", ipNet.String()) // #nosec G204

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("arp-scan failed: %w", err)
	}

	// Parse output
	return ns.parseArpScanOutput(string(output)), nil
}

// scanWithArp performs basic ARP scan using system ARP table.
func (ns *NetworkScanner) scanWithArp(ctx context.Context, ipNet *net.IPNet) ([]*ScanResult, error) {
	// Get ARP table
	arpResults, err := ns.getArpTable(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get ARP table: %w", err)
	}

	// Filter results by network
	var results []*ScanResult

	for _, result := range arpResults {
		if ipNet.Contains(net.ParseIP(result.IP)) {
			results = append(results, result)
		}
	}

	return results, nil
}

// scanBasic performs basic network scan using ping sweep.
func (ns *NetworkScanner) scanBasic(ctx context.Context, ipNet *net.IPNet) []*ScanResult {
	// Get network IPs
	ips := ns.getNetworkIPs(ipNet)

	// Create channel for results
	results := make(chan *ScanResult, len(ips))

	var wg sync.WaitGroup

	// Limit concurrent pings
	semaphore := make(chan struct{}, maxConcurrentPings)

	// Ping each IP
	for _, ip := range ips {
		wg.Add(1)

		go func(ip string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}

			defer func() { <-semaphore }()

			// Ping host
			if ns.pingHost(ctx, ip) {
				// Try to get hostname
				hostname := ns.getHostname(ctx, ip)

				results <- &ScanResult{
					IP:       ip,
					Hostname: hostname,
					MAC:      "", // MAC not available with ping
					Vendor:   "",
					Status:   "online",
				}
			}
		}(ip)
	}

	// Wait for all pings to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	scanResults := make([]*ScanResult, 0, len(ips))
	for result := range results {
		scanResults = append(scanResults, result)
	}

	return scanResults
}

// pingHost pings a single host.
func (ns *NetworkScanner) pingHost(ctx context.Context, ip string) bool {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	// Run ping command
	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "1", ip)
	err := cmd.Run()

	return err == nil
}

// getNetworkIPs returns all IP addresses in the network range.
func (ns *NetworkScanner) getNetworkIPs(ipNet *net.IPNet) []string {
	var ips []string

	// Get network address and mask
	ip := ipNet.IP.To4()
	if ip == nil {
		return ips
	}

	mask := ipNet.Mask
	if len(mask) != ipv4AddressLength {
		return ips
	}

	// Calculate network and broadcast addresses
	network := make(net.IP, ipv4AddressLength)
	broadcast := make(net.IP, ipv4AddressLength)

	for i := range ipv4AddressLength {
		network[i] = ip[i] & mask[i]
		broadcast[i] = ip[i] | (mask[i] ^ broadcastMask)
	}

	// Generate all IPs in range
	current := make(net.IP, ipv4AddressLength)
	copy(current, network)
	ns.incrementIP(current) // Skip network address

	for !current.Equal(broadcast) {
		// Check if we've reached broadcast address
		ips = append(ips, current.String())

		// Increment IP
		ns.incrementIP(current)
	}

	return ips
}

// incrementIP increments an IP address by 1.
func (ns *NetworkScanner) incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

// getArpTable gets the system ARP table.
func (ns *NetworkScanner) getArpTable(ctx context.Context) ([]*ScanResult, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, hostnameTimeout)
	defer cancel()

	// Run arp command
	cmd := exec.CommandContext(ctx, "arp", "-a")

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("arp command failed: %w", err)
	}

	// Parse output
	return ns.parseArpOutput(string(output)), nil
}

// parseNmapOutput parses nmap output and returns scan results.
func (ns *NetworkScanner) parseNmapOutput(output string) []*ScanResult {
	var results []*ScanResult

	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for "Nmap scan report" lines
		if strings.Contains(line, "Nmap scan report") {
			// Extract IP address
			parts := strings.Fields(line)
			if len(parts) >= arpTableMinParts {
				ip := parts[4]
				if net.ParseIP(ip) != nil {
					results = append(results, &ScanResult{
						IP:       ip,
						Hostname: "",
						MAC:      "",
						Vendor:   "",
						Status:   "online",
					})
				}
			}
		}
	}

	return results
}

// parseArpScanOutput parses arp-scan output and returns scan results.
func (ns *NetworkScanner) parseArpScanOutput(output string) []*ScanResult {
	var results []*ScanResult

	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Interface:") || strings.HasPrefix(line, "Starting") {
			continue
		}

		// Parse line: IP MAC VENDOR
		parts := strings.Fields(line)
		if len(parts) >= arpScanMinParts {
			ip := parts[0]
			mac := parts[1]

			// Validate IP and MAC
			if net.ParseIP(ip) != nil && ns.isValidMAC(mac) {
				vendor := ""
				if len(parts) >= arpScanVendorParts {
					vendor = strings.Join(parts[2:], " ")
				}

				results = append(results, &ScanResult{
					IP:       ip,
					Hostname: "",
					MAC:      mac,
					Vendor:   vendor,
					Status:   "online",
				})
			}
		}
	}

	return results
}

// parseArpOutput parses arp -a output and returns scan results.
func (ns *NetworkScanner) parseArpOutput(output string) []*ScanResult {
	var results []*ScanResult

	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse line: HOSTNAME (IP) at MAC [TYPE] on INTERFACE
		// Example: ? (192.168.1.1) at aa:bb:cc:dd:ee:ff [ether] on en0
		parts := strings.Fields(line)
		if len(parts) >= arpMinParts {
			ns.processArpLine(parts, &results)
		}
	}

	return results
}

// hasCommand checks if a command is available in PATH.
func (ns *NetworkScanner) hasCommand(cmd string) bool {
	_, err := exec.LookPath(cmd)

	return err == nil
}

// isValidMAC validates MAC address format.
func (ns *NetworkScanner) isValidMAC(mac string) bool {
	// Check if MAC contains colons and has correct length
	if len(mac) != macAddressLength {
		return false
	}

	parts := strings.Split(mac, ":")
	if len(parts) != macAddressParts {
		return false
	}

	// Check each part is valid hex
	for _, part := range parts {
		if len(part) != macPartLength {
			return false
		}

		if _, err := strconv.ParseUint(part, 16, 8); err != nil {
			return false
		}
	}

	return true
}

// getHostname tries to resolve hostname for an IP address.
func (ns *NetworkScanner) getHostname(ctx context.Context, ip string) string {
	// Try reverse DNS lookup
	resolver := &net.Resolver{}

	names, err := resolver.LookupAddr(ctx, ip)
	if err != nil || len(names) == 0 {
		return ""
	}

	// Return the first name, remove trailing dot
	hostname := names[0]
	if len(hostname) > 0 && hostname[len(hostname)-1] == '.' {
		hostname = hostname[:len(hostname)-1]
	}

	return hostname
}

// processArpLine processes a single ARP table line and adds valid entries to results.
func (ns *NetworkScanner) processArpLine(parts []string, results *[]*ScanResult) {
	// Extract IP from (IP) format
	ipPart := parts[1]
	if !strings.HasPrefix(ipPart, "(") || !strings.HasSuffix(ipPart, ")") {
		return
	}

	ip := ipPart[1 : len(ipPart)-1]
	mac := parts[3]

	// Validate IP and MAC
	if net.ParseIP(ip) == nil || !ns.isValidMAC(mac) {
		return
	}

	hostname := parts[0]
	if hostname == "?" {
		hostname = ""
	}

	*results = append(*results, &ScanResult{
		IP:       ip,
		Hostname: hostname,
		MAC:      mac,
		Vendor:   "",
		Status:   "online",
	})
}
