package lanresolver

import (
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Lease represents a DHCP lease entry.
type Lease struct {
	Expire   time.Time `json:"expire"`
	MAC      string    `json:"mac"`
	IP       string    `json:"ip"`
	Hostname string    `json:"hostname"`
	ID       string    `json:"id"`
}

// FileReader interface for reading files (for testing).
type FileReader interface {
	ReadFile(path string) ([]byte, error)
}

// OSFileReader implements FileReader using real OS files.
type OSFileReader struct{}

func (r *OSFileReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec
}

// LeaseManager manages DHCP lease entries.
type LeaseManager struct {
	leasesPath string
	leases     map[string]*Lease // MAC -> lease (changed from hostname to support devices without hostname)
	mu         sync.RWMutex
	fileReader FileReader
}

// NewLeaseManager creates a new lease manager.
func NewLeaseManager(leasesPath string) *LeaseManager {
	return &LeaseManager{
		leasesPath: leasesPath,
		leases:     make(map[string]*Lease),
		fileReader: &OSFileReader{},
	}
}

// NewLeaseManagerWithReader creates a new lease manager with custom file reader.
func NewLeaseManagerWithReader(leasesPath string, fileReader FileReader) *LeaseManager {
	return &LeaseManager{
		leasesPath: leasesPath,
		leases:     make(map[string]*Lease),
		fileReader: fileReader,
	}
}

// SetLeaseFile updates the lease file path and reloads leases.
func (lm *LeaseManager) SetLeaseFile(leasesPath string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.leasesPath = leasesPath
	// Clear existing leases
	lm.leases = make(map[string]*Lease)

	// Reload leases from new file (without holding the lock)
	lm.mu.Unlock()
	err := lm.LoadLeases()
	lm.mu.Lock()

	// Ignore error - file might not exist yet
	// This is expected for OpenWrt and other systems
	_ = err
}

// LoadLeases loads leases from the DHCP leases file.
func (lm *LeaseManager) LoadLeases() error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Clear existing leases
	lm.leases = make(map[string]*Lease)

	content, err := lm.fileReader.ReadFile(lm.leasesPath)
	if err != nil {
		return err
	}

	lines := strings.SplitSeq(string(content), "\n")
	parsedCount := 0
	skippedCount := 0

	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		lease := lm.ParseLeaseLine(line)
		if lease != nil {
			// Use MAC as key to support devices without hostname (common on OpenWrt)
			// This allows us to store and retrieve leases even when hostname is empty or "*"
			if lease.MAC != "" {
				lm.leases[lease.MAC] = lease
				parsedCount++
			} else {
				skippedCount++
			}
		} else {
			skippedCount++
		}
	}

	return nil
}

// GetLease returns a lease by hostname (searches by hostname field).
func (lm *LeaseManager) GetLease(hostname string) *Lease {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// Search through all leases to find one with matching hostname
	for _, lease := range lm.leases {
		if lease.Hostname == hostname {
			// Check if lease is still valid
			if time.Now().Before(lease.Expire) {
				return lease
			}
		}
	}

	return nil
}

// GetLeaseByMAC returns a lease by MAC address.
func (lm *LeaseManager) GetLeaseByMAC(mac string) *Lease {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	lease, exists := lm.leases[mac]
	if !exists {
		return nil
	}

	// Check if lease is still valid
	if time.Now().After(lease.Expire) {
		return nil
	}

	return lease
}

// GetLeaseByIP returns a lease by IP address.
func (lm *LeaseManager) GetLeaseByIP(ip string) *Lease {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	// Search through all leases to find one with matching IP
	for _, lease := range lm.leases {
		if lease.IP == ip {
			// Check if lease is still valid
			if time.Now().Before(lease.Expire) {
				return lease
			}
		}
	}

	return nil
}

// GetAllLeases returns all valid leases.
func (lm *LeaseManager) GetAllLeases() []*Lease {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	validLeases := make([]*Lease, 0, len(lm.leases))

	now := time.Now()

	for _, lease := range lm.leases {
		if now.Before(lease.Expire) {
			validLeases = append(validLeases, lease)
		}
	}

	return validLeases
}

// GetLeasesPath returns the path to the leases file.
func (lm *LeaseManager) GetLeasesPath() string {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	return lm.leasesPath
}

// ResolveHostname resolves a hostname to IP addresses.
func (lm *LeaseManager) ResolveHostname(hostname string) ([]net.IP, []net.IP) {
	lease := lm.GetLease(hostname)
	if lease == nil {
		return nil, nil
	}

	ip := net.ParseIP(lease.IP)
	if ip == nil {
		return nil, nil
	}

	var (
		aRecords    []net.IP
		aaaaRecords []net.IP
	)

	if ip.To4() != nil {
		aRecords = append(aRecords, ip)
	} else {
		aaaaRecords = append(aaaaRecords, ip)
	}

	return aRecords, aaaaRecords
}

// IsValidHostname checks if a hostname has a valid lease.
func (lm *LeaseManager) IsValidHostname(hostname string) bool {
	return lm.GetLease(hostname) != nil
}

// GetLeaseCount returns the number of valid leases.
func (lm *LeaseManager) GetLeaseCount() int {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	count := 0
	now := time.Now()

	for _, lease := range lm.leases {
		if now.Before(lease.Expire) {
			count++
		}
	}

	return count
}

// ParseLeaseLine parses a single line from the DHCP leases file.
// Format: <expire> <mac> <ip> <hostname> <id> (OpenWrt format).
// Hostname can be empty, "*", or missing - we handle all cases.
// ParseLeaseLine parses a single lease line (public for testing).
func (lm *LeaseManager) ParseLeaseLine(line string) *Lease {
	parts := strings.Fields(line)

	const minLeaseParts = 3 // MAC, IP, and expire are required
	if len(parts) < minLeaseParts {
		return nil
	}

	// Parse expire time (Unix timestamp)
	expireInt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil
	}

	expire := time.Unix(expireInt, 0)

	// Check if lease is expired
	if time.Now().After(expire) {
		return nil
	}

	lease := &Lease{
		Expire: expire,
		MAC:    parts[1],
		IP:     parts[2],
	}

	// Hostname is optional (can be empty or "*" on OpenWrt)
	if len(parts) > 3 {
		hostname := parts[3]
		// Normalize empty hostname or "*" to empty string
		if hostname != "*" && hostname != "" {
			lease.Hostname = hostname
		}
	}

	// ID is optional
	if len(parts) > 4 {
		lease.ID = parts[4]
	}

	return lease
}
