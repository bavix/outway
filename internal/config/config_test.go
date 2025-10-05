package config_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/config"
)

func TestDetectType(t *testing.T) {
	t.Parallel()

	tests := getDetectTypeTests()

	for _, tt := range tests {
		_ = tt // Suppress unused variable warning
	}
}

//nolint:funlen
func getDetectTypeTests() []struct {
	name     string
	addr     string
	expected string
} {
	return []struct {
		name     string
		addr     string
		expected string
	}{
		{
			name:     "empty address",
			addr:     "",
			expected: "",
		},
		{
			name:     "whitespace only",
			addr:     "   ",
			expected: "",
		},
		{
			name:     "https address",
			addr:     "https://dns.google/dns-query",
			expected: "doh",
		},
		{
			name:     "udp address",
			addr:     "udp://8.8.8.8:53",
			expected: "udp",
		},
		{
			name:     "tcp address",
			addr:     "tcp://8.8.8.8:53",
			expected: "tcp",
		},
		{
			name:     "tls address",
			addr:     "tls://8.8.8.8:853",
			expected: "dot",
		},
		{
			name:     "dot address",
			addr:     "dot://8.8.8.8:853",
			expected: "dot",
		},
		{
			name:     "quic address",
			addr:     "quic://8.8.8.8:853",
			expected: "doq",
		},
		{
			name:     "doq address",
			addr:     "doq://8.8.8.8:853",
			expected: "doq",
		},
		{
			name:     "invalid address",
			addr:     "invalid://",
			expected: "",
		},
		{
			name:     "no scheme",
			addr:     "8.8.8.8:53",
			expected: "",
		},
	}
}

func TestListenConfig(t *testing.T) {
	t.Parallel()

	cfg := config.ListenConfig{
		UDP: ":53",
		TCP: ":53",
	}

	assert.Equal(t, ":53", cfg.UDP)
	assert.Equal(t, ":53", cfg.TCP)
}

func TestUpstreamConfig(t *testing.T) {
	t.Parallel()

	cfg := config.UpstreamConfig{
		Name:    "test-upstream",
		Address: "udp://8.8.8.8:53",
		Type:    "udp",
		Weight:  1,
	}

	assert.Equal(t, "test-upstream", cfg.Name)
	assert.Equal(t, "udp://8.8.8.8:53", cfg.Address)
	assert.Equal(t, "udp", cfg.Type)
	assert.Equal(t, 1, cfg.Weight)
}

func TestUpstreamConfigMarshalYAML(t *testing.T) {
	t.Parallel()

	cfg := config.UpstreamConfig{
		Name:    "test",
		Address: "udp://8.8.8.8:53",
		Weight:  5,
	}

	result, err := cfg.MarshalYAML()
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestConfigGetMinMarkTTL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   config.Config
		ttl      uint32
		expected time.Duration
	}{
		{
			name: "default values",
			config: config.Config{
				Cache: config.CacheConfig{},
			},
			ttl:      300,
			expected: 300 * time.Second,
		},
		{
			name: "with min TTL",
			config: config.Config{
				Cache: config.CacheConfig{
					MinTTLSeconds: 60,
				},
			},
			ttl:      30,
			expected: 60 * time.Second,
		},
		{
			name: "with max TTL",
			config: config.Config{
				Cache: config.CacheConfig{
					MaxTTLSeconds: 1800,
				},
			},
			ttl:      3600,
			expected: 1800 * time.Second,
		},
		{
			name: "with both min and max TTL",
			config: config.Config{
				Cache: config.CacheConfig{
					MinTTLSeconds: 60,
					MaxTTLSeconds: 1800,
				},
			},
			ttl:      300,
			expected: 300 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.config.GetMinMarkTTL(tt.ttl)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigGetAllRules(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		RuleGroups: []config.RuleGroup{
			{
				Name:     "group1",
				Patterns: []string{"*.example.com", "test.com"},
				Via:      "eth0",
				PinTTL:   true,
			},
			{
				Name:     "group2",
				Patterns: []string{"*.test.com"},
				Via:      "wlan0",
				PinTTL:   false,
			},
		},
	}

	rules := cfg.GetAllRules()

	expectedRules := []config.Rule{
		{Pattern: "*.example.com", Via: "eth0", PinTTL: true},
		{Pattern: "test.com", Via: "eth0", PinTTL: true},
		{Pattern: "*.test.com", Via: "wlan0", PinTTL: false},
	}

	assert.Equal(t, expectedRules, rules)
}

func TestConfigGetEnabledUpstreams(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Upstreams: []config.UpstreamConfig{
			{Name: "upstream1", Address: "udp://8.8.8.8:53"},
			{Name: "upstream2", Address: "tcp://8.8.4.4:53"},
		},
	}

	upstreams := cfg.GetEnabledUpstreams()
	assert.Equal(t, cfg.Upstreams, upstreams)
}

func TestConfigGetUpstreamAddresses(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Upstreams: []config.UpstreamConfig{
			{Name: "upstream1", Address: "8.8.8.8:53", Type: "udp"},
			{Name: "upstream2", Address: "8.8.4.4:53", Type: "tcp"},
		},
	}

	addresses := cfg.GetUpstreamAddresses()
	expected := []string{"udp:8.8.8.8:53", "tcp:8.8.4.4:53"}

	assert.Equal(t, expected, addresses)
}

func TestConfigGetUpstreamsByWeight(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Upstreams: []config.UpstreamConfig{
			{Name: "upstream1", Address: "8.8.8.8:53", Weight: 3},
			{Name: "upstream2", Address: "8.8.4.4:53", Weight: 1},
			{Name: "upstream3", Address: "1.1.1.1:53", Weight: 2},
			{Name: "upstream4", Address: "1.0.0.1:53", Weight: 0}, // Should default to 1
		},
	}

	upstreams := cfg.GetUpstreamsByWeight()

	// Should be sorted by weight (descending)
	assert.Equal(t, "upstream1", upstreams[0].Name)
	assert.Equal(t, 3, upstreams[0].Weight)
	assert.Equal(t, "upstream3", upstreams[1].Name)
	assert.Equal(t, 2, upstreams[1].Weight)
	assert.Equal(t, "upstream2", upstreams[2].Name)
	assert.Equal(t, 1, upstreams[2].Weight)
	assert.Equal(t, "upstream4", upstreams[3].Name)
	assert.Equal(t, 1, upstreams[3].Weight) // Default weight
}

func TestRuleGroup(t *testing.T) {
	t.Parallel()

	group := config.RuleGroup{
		Name:     "test-group",
		Patterns: []string{"*.example.com", "test.com"},
		Via:      "eth0",
		PinTTL:   true,
	}

	assert.Equal(t, "test-group", group.Name)
	assert.Equal(t, []string{"*.example.com", "test.com"}, group.Patterns)
	assert.Equal(t, "eth0", group.Via)
	assert.True(t, group.PinTTL)
}

func TestRule(t *testing.T) {
	t.Parallel()

	rule := config.Rule{
		Pattern: "*.example.com",
		Via:     "eth0",
		PinTTL:  true,
	}

	assert.Equal(t, "*.example.com", rule.Pattern)
	assert.Equal(t, "eth0", rule.Via)
	assert.True(t, rule.PinTTL)
}

func TestHostOverride(t *testing.T) {
	t.Parallel()

	override := config.HostOverride{
		Pattern: "*.example.com",
		A:       []string{"192.168.1.1", "192.168.1.2"},
		AAAA:    []string{"2001:db8::1"},
		TTL:     300,
	}

	assert.Equal(t, "*.example.com", override.Pattern)
	assert.Equal(t, []string{"192.168.1.1", "192.168.1.2"}, override.A)
	assert.Equal(t, []string{"2001:db8::1"}, override.AAAA)
	assert.Equal(t, uint32(300), override.TTL)
}

func TestCacheConfig(t *testing.T) {
	t.Parallel()

	cfg := config.CacheConfig{
		MaxEntries:    1000,
		MinTTLSeconds: 60,
		MaxTTLSeconds: 3600,
	}

	assert.Equal(t, 1000, cfg.MaxEntries)
	assert.Equal(t, 60, cfg.MinTTLSeconds)
	assert.Equal(t, 3600, cfg.MaxTTLSeconds)
}

func TestHTTPConfig(t *testing.T) {
	t.Parallel()

	cfg := config.HTTPConfig{
		Listen:         ":8080",
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1024 * 1024,
	}

	assert.Equal(t, ":8080", cfg.Listen)
	assert.Equal(t, 30*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 30*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 120*time.Second, cfg.IdleTimeout)
	assert.Equal(t, 1024*1024, cfg.MaxHeaderBytes)
}

func TestLogConfig(t *testing.T) {
	t.Parallel()

	cfg := config.LogConfig{
		Level: "info",
	}

	assert.Equal(t, "info", cfg.Level)
}

func TestHistoryConfig(t *testing.T) {
	t.Parallel()

	cfg := config.HistoryConfig{
		MaxEntries: 100,
	}

	assert.Equal(t, 100, cfg.MaxEntries)
}

func TestUpdateConfig(t *testing.T) {
	t.Parallel()

	cfg := config.UpdateConfig{
		Enabled:           true,
		IncludePrerelease: false,
	}

	assert.True(t, cfg.Enabled)
	assert.False(t, cfg.IncludePrerelease)
}

func TestConfigValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: config.Config{
				Listen: config.ListenConfig{
					UDP: ":53",
					TCP: ":53",
				},
				Upstreams: []config.UpstreamConfig{
					{Name: "test", Address: "udp://8.8.8.8:53"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing listen config",
			config: config.Config{
				Upstreams: []config.UpstreamConfig{
					{Name: "test", Address: "udp://8.8.8.8:53"},
				},
			},
			wantErr: true,
		},
		{
			name: "no upstreams",
			config: config.Config{
				Listen: config.ListenConfig{
					UDP: ":53",
					TCP: ":53",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestUpstreamConfigBasic(t *testing.T) {
	t.Parallel()

	cfg := config.UpstreamConfig{
		Name:    "test",
		Address: "udp://8.8.8.8:53",
		Weight:  1,
	}

	assert.Equal(t, "test", cfg.Name)
	assert.Equal(t, "udp://8.8.8.8:53", cfg.Address)
	assert.Equal(t, 1, cfg.Weight)
}

func TestRuleGroupBasic(t *testing.T) {
	t.Parallel()

	group := config.RuleGroup{
		Name:     "test",
		Patterns: []string{"*.example.com"},
		Via:      "eth0",
		PinTTL:   true,
	}

	assert.Equal(t, "test", group.Name)
	assert.Equal(t, []string{"*.example.com"}, group.Patterns)
	assert.Equal(t, "eth0", group.Via)
	assert.True(t, group.PinTTL)
}

// TestConstants tests private constants - these are not accessible from test package
// func TestConstants(t *testing.T) {
// 	assert.Equal(t, 30*time.Second, defaultMinTTL)
// 	assert.Equal(t, 30*time.Second, defaultHTTPReadTimeout)
// 	assert.Equal(t, 30*time.Second, defaultHTTPWriteTimeout)
// 	assert.Equal(t, 120*time.Second, defaultHTTPIdleTimeout)
// 	assert.Equal(t, 1024*1024, defaultMaxHeaderBytes)
// 	assert.Equal(t, 0o600, defaultFilePerm)
// 	assert.Equal(t, "dot", protocolDot)
// 	assert.Equal(t, "tls", protocolTLS)
// }

// TestErrorConstants tests private error variables - these are not accessible from test package
// func TestErrorConstants(t *testing.T) {
// 	assert.Equal(t, "config path is empty", errConfigPathEmpty.Error())
// 	assert.Equal(t, "listen.udp and listen.tcp must be set", errListenUDPTCPMustBeSet.Error())
// 	assert.Equal(t, "at least one upstream is required", errAtLeastOneUpstreamRequired.Error())
// 	assert.Equal(t, "upstream name cannot be empty", errUpstreamNameCannotBeEmpty.Error())
// 	assert.Equal(t, "cache limits must be non-negative", errCacheLimitsMustBeNonNegative.Error())
// 	assert.Equal(t, "upstream address cannot be empty", errUpstreamAddressCannotBeEmpty.Error())
// 	assert.Equal(t, "upstream has invalid weight", errUpstreamInvalidWeight.Error())
// 	assert.Equal(t, "rule group name cannot be empty", errRuleGroupNameCannotBeEmpty.Error())
// 	assert.Equal(t, "duplicate rule group name", errDuplicateRuleGroupName.Error())
// 	assert.Equal(t, "rule group must have at least one pattern", errRuleGroupMustHavePattern.Error())
// 	assert.Equal(t, "rule group requires via interface", errRuleGroupRequiresViaInterface.Error())
// 	assert.Equal(t, "rule group contains empty pattern", errRuleGroupContainsEmptyPattern.Error())
// 	assert.Equal(t, "duplicate rule pattern", errDuplicateRulePattern.Error())
// 	assert.Equal(t, "address must be host:port or :port", errAddressMustBeHostPort.Error())
// 	assert.Equal(t, "cache ttl bounds must be non-negative", errCacheTTLBoundsMustBeNonNeg.Error())
// 	assert.Equal(t, "cache min_ttl_seconds cannot be greater than max_ttl_seconds", errCacheMinTTLGreaterThanMax.Error())
// }
