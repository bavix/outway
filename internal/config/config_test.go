package config

import (
	"os"
	"testing"
)

// Test detectType function with various address formats
func TestDetectType(t *testing.T) {
	tests := []struct {
		addr     string
		expected string
	}{
		// Plain addresses should default to UDP
		{"1.1.1.1:53", "udp"},
		{"8.8.8.8:53", "udp"},
		{"[2606:4700:4700::1111]:53", "udp"},
		
		// Port 853 should be detected as DoT
		{"1.1.1.1:853", "dot"},
		{"cloudflare-dns.com:853", "dot"},
		
		// Explicit URL schemes
		{"udp://1.1.1.1:53", "udp"},
		{"tcp://1.1.1.1:53", "tcp"},
		{"https://cloudflare-dns.com/dns-query", "doh"},
		{"tls://1.1.1.1:853", "dot"},
		{"dot://1.1.1.1:853", "dot"},
		{"quic://1.1.1.1:853", "doq"},
		{"doq://1.1.1.1:853", "doq"},
		
		// Edge cases
		{"", ""},
		{"  ", ""},
	}
	
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			result := detectType(tt.addr)
			if result != tt.expected {
				t.Errorf("detectType(%q) = %q, want %q", tt.addr, result, tt.expected)
			}
		})
	}
}

// Test that upstreams preserve their types after config save/reload
func TestUpstreamTypePersistence(t *testing.T) {
	// Create a config with various upstream formats
	testConfig := `app_name: outway
listen:
  udp: :5353
  tcp: :5353
upstreams:
- name: plain-address
  address: 1.1.1.1:53
  weight: 1
- name: dot-port
  address: 1.1.1.1:853
  weight: 1
- name: explicit-udp
  address: udp://8.8.8.8:53
  weight: 1
- name: doh
  address: https://cloudflare-dns.com/dns-query
  weight: 1
rule_groups:
- name: default
  via: default
  patterns:
  - "*"
`
	
	tmpfile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	
	if _, err := tmpfile.Write([]byte(testConfig)); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()
	
	// Load config
	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	
	// Verify all upstreams have types
	expectedTypes := map[string]string{
		"plain-address":  "udp",
		"dot-port":       "dot",
		"explicit-udp":   "udp",
		"doh":            "doh",
	}
	
	for _, u := range cfg.Upstreams {
		if expected, ok := expectedTypes[u.Name]; ok {
			if u.Type != expected {
				t.Errorf("Upstream %s: got type %q, want %q", u.Name, u.Type, expected)
			}
		}
		
		if u.Type == "" {
			t.Errorf("Upstream %s has empty type after initial load", u.Name)
		}
	}
	
	// Save config
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	
	// Reload config
	cfg2, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}
	
	// Verify types are still correct after reload
	for _, u := range cfg2.Upstreams {
		if expected, ok := expectedTypes[u.Name]; ok {
			if u.Type != expected {
				t.Errorf("After reload - Upstream %s: got type %q, want %q", u.Name, u.Type, expected)
			}
		}
		
		if u.Type == "" {
			t.Errorf("Upstream %s has empty type after reload", u.Name)
		}
	}
}
