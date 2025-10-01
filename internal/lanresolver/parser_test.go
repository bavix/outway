package lanresolver

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseLeases(t *testing.T) {
	// Create temp file with lease data
	tmpDir := t.TempDir()
	leasesFile := filepath.Join(tmpDir, "dhcp.leases")

	content := `1633024800 aa:bb:cc:dd:ee:ff 192.168.1.100 myhost *
1633024900 11:22:33:44:55:66 192.168.1.101 router id-123
1633025000 aa:aa:bb:bb:cc:cc 2001:db8::1 ipv6host *
1633025100 ff:ff:ff:ff:ff:ff 192.168.1.102 * *
# Comment line
invalid line without enough fields
`

	if err := os.WriteFile(leasesFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	leases, err := ParseLeases(leasesFile)
	if err != nil {
		t.Fatal(err)
	}

	// Should have 3 valid leases (4th has * hostname, 5th and 6th are invalid)
	if len(leases) != 3 {
		t.Errorf("expected 3 leases, got %d", len(leases))
	}

	// Check first lease
	if leases[0].Hostname != "myhost" {
		t.Errorf("expected hostname 'myhost', got '%s'", leases[0].Hostname)
	}
	if leases[0].IP != "192.168.1.100" {
		t.Errorf("expected IP '192.168.1.100', got '%s'", leases[0].IP)
	}
	if leases[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("expected MAC 'aa:bb:cc:dd:ee:ff', got '%s'", leases[0].MAC)
	}
	expectedTime := time.Unix(1633024800, 0)
	if !leases[0].ExpiresAt.Equal(expectedTime) {
		t.Errorf("expected expire time %v, got %v", expectedTime, leases[0].ExpiresAt)
	}

	// Check second lease with ID
	if leases[1].ID != "id-123" {
		t.Errorf("expected ID 'id-123', got '%s'", leases[1].ID)
	}
}

func TestBuildHostMap(t *testing.T) {
	leases := []Lease{
		{Hostname: "myhost", IP: "192.168.1.100", MAC: "aa:bb:cc:dd:ee:ff"},
		{Hostname: "router", IP: "192.168.1.1", MAC: "11:22:33:44:55:66"},
		{Hostname: "server", IP: "192.168.1.10", MAC: "aa:aa:bb:bb:cc:cc"},
		{Hostname: "server", IP: "2001:db8::10", MAC: "aa:aa:bb:bb:cc:cc"}, // Same host, IPv6
	}

	zones := []string{"lan", "home"}

	hostMap := BuildHostMap(leases, zones)

	// Check myhost.lan
	ips, ok := hostMap["myhost.lan"]
	if !ok {
		t.Error("expected 'myhost.lan' in host map")
	}
	if len(ips) != 1 || ips[0] != "192.168.1.100" {
		t.Errorf("expected IPs [192.168.1.100], got %v", ips)
	}

	// Check myhost.home
	ips, ok = hostMap["myhost.home"]
	if !ok {
		t.Error("expected 'myhost.home' in host map")
	}
	if len(ips) != 1 || ips[0] != "192.168.1.100" {
		t.Errorf("expected IPs [192.168.1.100], got %v", ips)
	}

	// Check bare hostname
	ips, ok = hostMap["myhost"]
	if !ok {
		t.Error("expected 'myhost' (bare) in host map")
	}
	if len(ips) != 1 || ips[0] != "192.168.1.100" {
		t.Errorf("expected IPs [192.168.1.100], got %v", ips)
	}

	// Check server with multiple IPs
	ips, ok = hostMap["server.lan"]
	if !ok {
		t.Error("expected 'server.lan' in host map")
	}
	if len(ips) != 2 {
		t.Errorf("expected 2 IPs for server.lan, got %d: %v", len(ips), ips)
	}
}

func TestParseLeasesNonExistent(t *testing.T) {
	_, err := ParseLeases("/nonexistent/file")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
