//nolint:paralleltest
package lanresolver_test

import (
	"testing"
	"testing/fstest"
	"time"

	"github.com/bavix/outway/internal/lanresolver"
)

func TestLeaseManager_ParseLeaseLine(t *testing.T) { //nolint:gocognit,cyclop,funlen
	lm := lanresolver.NewLeaseManager("")

	tests := []struct {
		name     string
		line     string
		expected *lanresolver.Lease
		valid    bool
	}{
		{
			name:  "valid lease",
			line:  "2000000000 00:11:22:33:44:55 192.168.1.100 host1.lan *",
			valid: true,
			expected: &lanresolver.Lease{
				Expire:   time.Unix(2000000000, 0),
				MAC:      "00:11:22:33:44:55",
				IP:       "192.168.1.100",
				Hostname: "host1.lan",
				ID:       "*",
			},
		},
		{
			name:  "valid lease with different values",
			line:  "2000000001 aa:bb:cc:dd:ee:ff 10.0.0.50 myhost.home dhcp-123",
			valid: true,
			expected: &lanresolver.Lease{
				Expire:   time.Unix(2000000001, 0),
				MAC:      "aa:bb:cc:dd:ee:ff",
				IP:       "10.0.0.50",
				Hostname: "myhost.home",
				ID:       "dhcp-123",
			},
		},
		{
			name:  "invalid lease - too few fields",
			line:  "1678886400 00:11:22:33:44:55 192.168.1.100",
			valid: false,
		},
		{
			name:  "invalid lease - empty line",
			line:  "",
			valid: false,
		},
		{
			name:  "invalid lease - comment",
			line:  "# This is a comment",
			valid: false,
		},
		{
			name:  "invalid lease - invalid timestamp",
			line:  "invalid 00:11:22:33:44:55 192.168.1.100 host1.lan *",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lease := lm.ParseLeaseLine(tt.line)

			if tt.valid { //nolint:nestif
				if lease == nil {
					t.Fatalf("Expected valid lease, got nil")
				}

				if tt.expected != nil {
					if lease.Expire.Unix() != tt.expected.Expire.Unix() {
						t.Errorf("Expected expires %v, got %v", tt.expected.Expire, lease.Expire)
					}

					if lease.MAC != tt.expected.MAC {
						t.Errorf("Expected MAC %s, got %s", tt.expected.MAC, lease.MAC)
					}

					if lease.IP != tt.expected.IP {
						t.Errorf("Expected IP %s, got %s", tt.expected.IP, lease.IP)
					}

					if lease.Hostname != tt.expected.Hostname {
						t.Errorf("Expected hostname %s, got %s", tt.expected.Hostname, lease.Hostname)
					}

					if lease.ID != tt.expected.ID {
						t.Errorf("Expected ID %s, got %s", tt.expected.ID, lease.ID)
					}
				}
			} else if lease != nil {
				t.Errorf("Expected invalid lease, got %+v", lease)
			}
		})
	}
}

func TestLeaseManager_LoadLeasesFromFS(t *testing.T) {
	// Create test filesystem using fstest.MapFS
	fs := fstest.MapFS{
		"dhcp.leases": &fstest.MapFile{
			Data: []byte(`2000000000 00:11:22:33:44:55 192.168.1.100 host1.lan *
2000000001 aa:bb:cc:dd:ee:ff 10.0.0.50 myhost.home dhcp-123
# This is a comment
2000000002 11:22:33:44:55:66 172.16.0.10 test.local *
`),
		},
	}

	lm := lanresolver.NewLeaseManagerWithReader("dhcp.leases", &FSTestFileReader{fs: fs})

	err := lm.LoadLeases()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that leases were loaded
	expectedCount := 3
	if lm.GetLeaseCount() != expectedCount {
		t.Fatalf("Expected %d leases, got %d", expectedCount, lm.GetLeaseCount())
	}

	// Check specific leases by resolving hostnames
	expectedLeases := map[string]string{
		"host1.lan":   "192.168.1.100",
		"myhost.home": "10.0.0.50",
		"test.local":  "172.16.0.10",
	}

	for hostname, expectedIP := range expectedLeases {
		lease := lm.GetLease(hostname)
		if lease == nil {
			t.Errorf("Expected lease for %s not found", hostname)

			continue
		}

		if lease.IP != expectedIP {
			t.Errorf("Expected IP %s for %s, got %s", expectedIP, hostname, lease.IP)
		}
	}
}

func TestLeaseManager_ResolveHostname(t *testing.T) { //nolint:funlen
	// Create test filesystem
	fs := fstest.MapFS{
		"dhcp.leases": &fstest.MapFile{
			Data: []byte(`2000000000 00:11:22:33:44:55 192.168.1.100 host1.lan *
2000000001 aa:bb:cc:dd:ee:ff 10.0.0.50 myhost.home dhcp-123
2000000002 11:22:33:44:55:66 172.16.0.10 test.local *`),
		},
	}

	lm := lanresolver.NewLeaseManagerWithReader("dhcp.leases", &FSTestFileReader{fs: fs})

	err := lm.LoadLeases()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	tests := []struct {
		name     string
		hostname string
		expected string
		found    bool
	}{
		{
			name:     "exact match",
			hostname: "host1.lan",
			expected: "192.168.1.100",
			found:    true,
		},
		{
			name:     "hostname only",
			hostname: "host1.lan",
			expected: "192.168.1.100",
			found:    true,
		},
		{
			name:     "different domain",
			hostname: "myhost.home",
			expected: "10.0.0.50",
			found:    true,
		},
		{
			name:     "not found",
			hostname: "nonexistent.lan",
			expected: "",
			found:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ips, _ := lm.ResolveHostname(tt.hostname)

			if tt.found {
				if len(ips) == 0 {
					t.Errorf("Expected IP %s for %s, got empty", tt.expected, tt.hostname)
				} else if ips[0].String() != tt.expected {
					t.Errorf("Expected IP %s for %s, got %s", tt.expected, tt.hostname, ips[0].String())
				}
			} else {
				if len(ips) > 0 {
					t.Errorf("Expected no IP for %s, got %s", tt.hostname, ips[0].String())
				}
			}
		})
	}
}

func TestLeaseManager_GetAllLeases(t *testing.T) {
	// Create test filesystem
	fs := fstest.MapFS{
		"dhcp.leases": &fstest.MapFile{
			Data: []byte(`2000000000 00:11:22:33:44:55 192.168.1.100 host1.lan *
2000000001 aa:bb:cc:dd:ee:ff 10.0.0.50 myhost.home dhcp-123`),
		},
	}

	lm := lanresolver.NewLeaseManagerWithReader("dhcp.leases", &FSTestFileReader{fs: fs})

	err := lm.LoadLeases()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	leases := lm.GetAllLeases()
	if len(leases) != 2 {
		t.Fatalf("Expected 2 leases, got %d", len(leases))
	}

	// Check that all leases are valid
	for _, lease := range leases {
		if lease.Hostname == "" {
			t.Error("Expected non-empty hostname")
		}

		if lease.IP == "" {
			t.Error("Expected non-empty IP")
		}

		if lease.MAC == "" {
			t.Error("Expected non-empty MAC")
		}
	}
}

func TestLeaseManager_GetLeaseCount(t *testing.T) {
	lm := lanresolver.NewLeaseManager("")

	// Initially empty
	if lm.GetLeaseCount() != 0 {
		t.Errorf("Expected 0 leases, got %d", lm.GetLeaseCount())
	}

	// Add some leases by loading from test data
	fs := fstest.MapFS{
		"dhcp.leases": &fstest.MapFile{
			Data: []byte(`2000000000 00:11:22:33:44:55 192.168.1.100 host1.lan *
2000000001 aa:bb:cc:dd:ee:ff 192.168.1.101 host2.lan *`),
		},
	}

	lm = lanresolver.NewLeaseManagerWithReader("dhcp.leases", &FSTestFileReader{fs: fs})

	err := lm.LoadLeases()
	if err != nil {
		t.Fatalf("Failed to load leases: %v", err)
	}

	if lm.GetLeaseCount() != 2 {
		t.Errorf("Expected 2 leases, got %d", lm.GetLeaseCount())
	}
}

func TestLeaseManager_CaseInsensitiveResolve(t *testing.T) {
	// Create test filesystem
	fs := fstest.MapFS{
		"dhcp.leases": &fstest.MapFile{
			Data: []byte(`2000000000 00:11:22:33:44:55 192.168.1.100 iPad.lan *`),
		},
	}

	lm := lanresolver.NewLeaseManagerWithReader("dhcp.leases", &FSTestFileReader{fs: fs})

	err := lm.LoadLeases()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test case-insensitive lookups
	tests := []struct {
		name     string
		hostname string
		expected string
		found    bool
	}{
		{
			name:     "original case",
			hostname: "ipad.lan",
			expected: "192.168.1.100",
			found:    true,
		},
		{
			name:     "uppercase",
			hostname: "IPAD.LAN",
			expected: "192.168.1.100",
			found:    true,
		},
		{
			name:     "mixed case",
			hostname: "iPAD.LAN",
			expected: "192.168.1.100",
			found:    true,
		},
		{
			name:     "lowercase",
			hostname: "ipad.lan",
			expected: "192.168.1.100",
			found:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ips, _ := lm.ResolveHostname(tt.hostname)

			if tt.found {
				if len(ips) == 0 {
					t.Errorf("Expected IP %s for %s, got empty", tt.expected, tt.hostname)
				} else if ips[0].String() != tt.expected {
					t.Errorf("Expected IP %s for %s, got %s", tt.expected, tt.hostname, ips[0].String())
				}
			} else {
				if len(ips) > 0 {
					t.Errorf("Expected no IP for %s, got %s", tt.hostname, ips[0].String())
				}
			}
		})
	}
}

// FSTestFileReader implements file reading using testing/fstest.MapFS.
type FSTestFileReader struct {
	fs fstest.MapFS
}

func (r *FSTestFileReader) ReadFile(path string) ([]byte, error) {
	file, err := r.fs.Open(path)
	if err != nil {
		return nil, err
	}

	defer func() { _ = file.Close() }()

	data := make([]byte, 0, 1024)

	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}

		if err != nil {
			break
		}
	}

	return data, nil
}
