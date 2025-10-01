//nolint:paralleltest
package localzone_test

import (
	"testing"
	"testing/fstest"

	"github.com/bavix/outway/internal/localzone"
)

func TestTestableZoneDetector_DetectZones_UCI(t *testing.T) {
	// Create test filesystem using fstest.MapFS
	fs := fstest.MapFS{
		"etc/config/dhcp": &fstest.MapFile{
			Data: []byte(`config dnsmasq 'lan'
	option domain 'lan'
	option local '/lan/'

config dnsmasq 'home'
	option domain 'home'
	option local '/home/'`),
		},
	}

	// Create mock file reader using fstest
	fileReader := &FSTestFileReader{fs: fs}

	config := localzone.DetectorConfig{
		UCIPath:           "etc/config/dhcp",
		ResolvPath:        "etc/resolv.conf",
		ManualZones:       []string{},
		DetectFromUCI:     true,
		DetectFromResolv:  false,
		DetectFromSystemd: false,
		DetectFromMDNS:    false,
	}

	detector := localzone.NewTestableZoneDetector(fileReader, config)

	zones, err := detector.DetectZones()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := []string{"lan", "home"}
	if len(zones) != len(expected) {
		t.Fatalf("Expected %d zones, got %d: %v", len(expected), len(zones), zones)
	}

	for _, expectedZone := range expected {
		found := false

		for _, zone := range zones {
			if zone == expectedZone {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("Expected zone %s not found in %v", expectedZone, zones)
		}
	}
}

func TestTestableZoneDetector_DetectZones_ResolvConf(t *testing.T) {
	fs := fstest.MapFS{
		"etc/resolv.conf": &fstest.MapFile{
			Data: []byte(`search lan home
domain lan`),
		},
	}

	fileReader := &FSTestFileReader{fs: fs}

	config := localzone.DetectorConfig{
		UCIPath:           "etc/config/dhcp",
		ResolvPath:        "etc/resolv.conf",
		ManualZones:       []string{},
		DetectFromUCI:     false,
		DetectFromResolv:  true,
		DetectFromSystemd: false,
		DetectFromMDNS:    false,
	}

	detector := localzone.NewTestableZoneDetector(fileReader, config)

	zones, err := detector.DetectZones()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := []string{"lan", "home"}
	if len(zones) != len(expected) {
		t.Fatalf("Expected %d zones, got %d: %v", len(expected), len(zones), zones)
	}
}

func TestTestableZoneDetector_DetectZones_Systemd(t *testing.T) {
	fs := fstest.MapFS{
		"run/systemd/resolve/resolv.conf": &fstest.MapFile{
			Data: []byte(`search local lan
domain local`),
		},
	}

	fileReader := &FSTestFileReader{fs: fs}

	config := localzone.DetectorConfig{
		UCIPath:           "etc/config/dhcp",
		ResolvPath:        "etc/resolv.conf",
		ManualZones:       []string{},
		DetectFromUCI:     false,
		DetectFromResolv:  false,
		DetectFromSystemd: true,
		DetectFromMDNS:    false,
	}

	detector := localzone.NewTestableZoneDetector(fileReader, config)

	zones, err := detector.DetectZones()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	t.Logf("Detected zones: %v", zones)

	expected := []string{}
	if len(zones) != len(expected) {
		t.Fatalf("Expected %d zones, got %d: %v", len(expected), len(zones), zones)
	}
}

func TestTestableZoneDetector_DetectZones_MDNS(t *testing.T) {
	fs := fstest.MapFS{
		"etc/avahi/avahi-daemon.conf": &fstest.MapFile{
			Data: []byte(`[server]
domain-name=local
host-name=test

[wide-area]
enable-wide-area=yes`),
		},
	}

	fileReader := &FSTestFileReader{fs: fs}

	config := localzone.DetectorConfig{
		UCIPath:           "etc/config/dhcp",
		ResolvPath:        "etc/resolv.conf",
		ManualZones:       []string{},
		DetectFromUCI:     false,
		DetectFromResolv:  false,
		DetectFromSystemd: false,
		DetectFromMDNS:    true,
	}

	detector := localzone.NewTestableZoneDetector(fileReader, config)

	zones, err := detector.DetectZones()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should fallback to common domains since mDNS parsing is simple
	expected := []string{}
	if len(zones) != len(expected) {
		t.Fatalf("Expected %d zones, got %d: %v", len(expected), len(zones), zones)
	}
}

func TestTestableZoneDetector_DetectZones_ManualZones(t *testing.T) {
	// No files exist
	fs := fstest.MapFS{}
	fileReader := &FSTestFileReader{fs: fs}

	config := localzone.DetectorConfig{
		UCIPath:           "etc/config/dhcp",
		ResolvPath:        "etc/resolv.conf",
		ManualZones:       []string{"custom.lan", "test.home"},
		DetectFromUCI:     true,
		DetectFromResolv:  true,
		DetectFromSystemd: true,
		DetectFromMDNS:    true,
	}

	detector := localzone.NewTestableZoneDetector(fileReader, config)

	zones, err := detector.DetectZones()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := []string{"custom.lan", "test.home"}
	if len(zones) != len(expected) {
		t.Fatalf("Expected %d zones, got %d: %v", len(expected), len(zones), zones)
	}
}

func TestTestableZoneDetector_DetectZones_Fallback(t *testing.T) {
	// No files exist and no manual zones
	fs := fstest.MapFS{}
	fileReader := &FSTestFileReader{fs: fs}

	config := localzone.DetectorConfig{
		UCIPath:           "etc/config/dhcp",
		ResolvPath:        "etc/resolv.conf",
		ManualZones:       []string{},
		DetectFromUCI:     true,
		DetectFromResolv:  true,
		DetectFromSystemd: true,
		DetectFromMDNS:    true,
	}

	detector := localzone.NewTestableZoneDetector(fileReader, config)

	zones, err := detector.DetectZones()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return empty array (no fallback)
	expected := []string{}
	if len(zones) != len(expected) {
		t.Fatalf("Expected %d zones, got %d: %v", len(expected), len(zones), zones)
	}
}

func TestTestableZoneDetector_DetectZones_Duplicates(t *testing.T) {
	fs := fstest.MapFS{
		"etc/config/dhcp": &fstest.MapFile{
			Data: []byte(`config dnsmasq 'lan'
	option domain 'lan'
	option local '/lan/'`),
		},
		"etc/resolv.conf": &fstest.MapFile{
			Data: []byte(`search lan home`),
		},
	}

	fileReader := &FSTestFileReader{fs: fs}

	config := localzone.DetectorConfig{
		UCIPath:           "etc/config/dhcp",
		ResolvPath:        "etc/resolv.conf",
		ManualZones:       []string{"lan"}, // Duplicate
		DetectFromUCI:     true,
		DetectFromResolv:  true,
		DetectFromSystemd: false,
		DetectFromMDNS:    false,
	}

	detector := localzone.NewTestableZoneDetector(fileReader, config)

	zones, err := detector.DetectZones()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should have no duplicates
	seen := make(map[string]bool)
	for _, zone := range zones {
		if seen[zone] {
			t.Errorf("Duplicate zone found: %s", zone)
		}

		seen[zone] = true
	}
}

// FSTestFileReader implements FileReader using testing/fstest.MapFS.
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

func (r *FSTestFileReader) FileExists(path string) bool {
	_, err := r.fs.Open(path)

	return err == nil
}
