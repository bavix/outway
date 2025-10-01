package localzone

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectLocalZonesOverride(t *testing.T) {
	// Override should take precedence
	zones := DetectLocalZones([]string{"custom", "override"}, "", "")

	if len(zones) != 2 {
		t.Errorf("expected 2 zones, got %d", len(zones))
	}
	if zones[0] != "custom" || zones[1] != "override" {
		t.Errorf("expected [custom, override], got %v", zones)
	}
}

func TestDetectFromUCI(t *testing.T) {
	tmpDir := t.TempDir()
	uciFile := filepath.Join(tmpDir, "dhcp")

	content := `
config dnsmasq
	option domain 'lan'
	option local '/home/'

config dnsmasq
	option domain 'office'
	option rebind_protection '1'
`

	if err := os.WriteFile(uciFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	zones := detectFromUCI(uciFile)

	if len(zones) != 3 {
		t.Errorf("expected 3 zones, got %d: %v", len(zones), zones)
	}

	// Check that zones are detected
	expected := map[string]bool{"lan": true, "home": true, "office": true}
	for _, z := range zones {
		if !expected[z] {
			t.Errorf("unexpected zone: %s", z)
		}
	}
}

func TestDetectFromResolv(t *testing.T) {
	tmpDir := t.TempDir()
	resolvFile := filepath.Join(tmpDir, "resolv.conf")

	content := `nameserver 192.168.1.1
search lan home
domain local
nameserver 8.8.8.8
`

	if err := os.WriteFile(resolvFile, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	zones := detectFromResolv(resolvFile)

	if len(zones) != 3 {
		t.Errorf("expected 3 zones, got %d: %v", len(zones), zones)
	}

	// Check that zones are detected
	expected := map[string]bool{"lan": true, "home": true, "local": true}
	for _, z := range zones {
		if !expected[z] {
			t.Errorf("unexpected zone: %s", z)
		}
	}
}

func TestCleanZones(t *testing.T) {
	input := []string{
		"LAN",
		"  home  ",
		"office.",
		".local.",
		"lan", // duplicate
		"",    // empty
	}

	result := cleanZones(input)

	if len(result) != 4 {
		t.Errorf("expected 4 zones, got %d: %v", len(result), result)
	}

	// Check normalization
	expected := map[string]bool{"lan": true, "home": true, "office": true, "local": true}
	for _, z := range result {
		if !expected[z] {
			t.Errorf("unexpected zone: %s", z)
		}
	}
}

func TestDetectLocalZonesPriority(t *testing.T) {
	tmpDir := t.TempDir()

	// Create UCI file
	uciFile := filepath.Join(tmpDir, "dhcp")
	uciContent := `
config dnsmasq
	option domain 'uci-lan'
`
	if err := os.WriteFile(uciFile, []byte(uciContent), 0o600); err != nil {
		t.Fatal(err)
	}

	// Create resolv file
	resolvFile := filepath.Join(tmpDir, "resolv.conf")
	resolvContent := `search resolv-lan`
	if err := os.WriteFile(resolvFile, []byte(resolvContent), 0o600); err != nil {
		t.Fatal(err)
	}

	// Test priority: override > UCI > resolv
	zones := DetectLocalZones([]string{"override-lan"}, uciFile, resolvFile)
	if len(zones) != 1 || zones[0] != "override-lan" {
		t.Errorf("override should take priority, got %v", zones)
	}

	zones = DetectLocalZones(nil, uciFile, resolvFile)
	if len(zones) != 1 || zones[0] != "uci-lan" {
		t.Errorf("UCI should take priority over resolv, got %v", zones)
	}

	zones = DetectLocalZones(nil, "/nonexistent", resolvFile)
	if len(zones) != 1 || zones[0] != "resolv-lan" {
		t.Errorf("resolv should be fallback, got %v", zones)
	}
}
