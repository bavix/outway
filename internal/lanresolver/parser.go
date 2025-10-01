package lanresolver

import (
	"bufio"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Lease represents a DHCP lease entry.
type Lease struct {
	Hostname  string    `json:"hostname"`
	IP        string    `json:"ip"`
	MAC       string    `json:"mac"`
	ExpiresAt time.Time `json:"expires_at"`
	ID        string    `json:"id,omitempty"`
}

// ParseLeases parses /tmp/dhcp.leases file.
// Format: <expire> <mac> <ip> <hostname> <id>
// Example: 1633024800 aa:bb:cc:dd:ee:ff 192.168.1.100 myhost *
func ParseLeases(path string) ([]Lease, error) {
	if path == "" {
		path = "/tmp/dhcp.leases"
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var leases []Lease
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 4 {
			continue // Skip invalid lines
		}

		// Parse expire time (unix timestamp)
		expireUnix, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			continue
		}

		lease := Lease{
			MAC:       parts[1],
			IP:        parts[2],
			Hostname:  parts[3],
			ExpiresAt: time.Unix(expireUnix, 0),
		}

		// Optional ID field
		if len(parts) >= 5 {
			lease.ID = parts[4]
		}

		// Validate IP address
		if net.ParseIP(lease.IP) == nil {
			continue
		}

		// Skip entries without hostname or with wildcard hostname
		if lease.Hostname == "" || lease.Hostname == "*" {
			continue
		}

		leases = append(leases, lease)
	}

	if err := scanner.Err(); err != nil {
		return leases, err
	}

	return leases, nil
}

// BuildHostMap creates a map of hostname.zone -> IP addresses.
func BuildHostMap(leases []Lease, zones []string) map[string][]string {
	hostMap := make(map[string][]string)

	for _, lease := range leases {
		// Add entries for each zone
		for _, zone := range zones {
			fqdn := strings.ToLower(lease.Hostname + "." + zone)
			hostMap[fqdn] = append(hostMap[fqdn], lease.IP)
		}

		// Also support bare hostname (without zone)
		bareHost := strings.ToLower(lease.Hostname)
		hostMap[bareHost] = append(hostMap[bareHost], lease.IP)
	}

	return hostMap
}
