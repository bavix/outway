package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	yaml "github.com/goccy/go-yaml"
)

var (
	errConfigPathEmpty               = errors.New("config path is empty")
	errListenUDPTCPMustBeSet         = errors.New("listen.udp and listen.tcp must be set")
	errAtLeastOneUpstreamRequired    = errors.New("at least one upstream is required")
	errUpstreamNameCannotBeEmpty     = errors.New("upstream name cannot be empty")
	errCacheLimitsMustBeNonNegative  = errors.New("cache limits must be non-negative")
	errUpstreamAddressCannotBeEmpty  = errors.New("upstream address cannot be empty")
	errUpstreamInvalidWeight         = errors.New("upstream has invalid weight")
	errRuleGroupNameCannotBeEmpty    = errors.New("rule group name cannot be empty")
	errDuplicateRuleGroupName        = errors.New("duplicate rule group name")
	errRuleGroupMustHavePattern      = errors.New("rule group must have at least one pattern")
	errRuleGroupRequiresViaInterface = errors.New("rule group requires via interface")
	errRuleGroupContainsEmptyPattern = errors.New("rule group contains empty pattern")
	errDuplicateRulePattern          = errors.New("duplicate rule pattern")
	errAddressMustBeHostPort         = errors.New("address must be host:port or :port")
	errCacheTTLBoundsMustBeNonNeg    = errors.New("cache ttl bounds must be non-negative")
	errCacheMinTTLGreaterThanMax     = errors.New("cache min_ttl_seconds cannot be greater than max_ttl_seconds")
)

const (
	defaultMinTTL           = 30 * time.Second
	defaultHTTPReadTimeout  = 30 * time.Second
	defaultHTTPWriteTimeout = 30 * time.Second
	defaultHTTPIdleTimeout  = 120 * time.Second
	defaultMaxHeaderBytes   = 1024 * 1024 // 1MB
	defaultFilePerm         = 0o600

	// Protocol constants.
	protocolDot = "dot"
	protocolTLS = "tls"
)

func detectType(addr string) string {
	a := strings.TrimSpace(addr)
	if a == "" {
		return ""
	}

	u, err := url.Parse(a)
	if err != nil || u.Scheme == "" {
		return ""
	}

	switch strings.ToLower(u.Scheme) {
	case "https":
		return "doh"
	case "udp":
		return "udp"
	case "tcp":
		return "tcp"
	case protocolTLS, protocolDot:
		return "dot"
	case "quic", "doq":
		return "doq"
	default:
		return ""
	}
}

// ListenConfig defines DNS server listening configuration.
type ListenConfig struct {
	UDP string `yaml:"udp"`
	TCP string `yaml:"tcp"`
}

// UpstreamConfig defines a DNS upstream server.
type UpstreamConfig struct {
	Name    string `json:"name"             yaml:"name"`
	Address string `json:"address"          yaml:"address"`
	Type    string `json:"type,omitempty"   yaml:"type,omitempty"` // optional; autodetected when empty
	Weight  int    `json:"weight,omitempty" yaml:"weight,omitempty"`
}

// MarshalYAML implements custom YAML marshaling for UpstreamConfig,
// omitting the derived Type field and normalizing weight.
func (u UpstreamConfig) MarshalYAML() (any, error) {
	type out struct {
		Name    string `yaml:"name"`
		Address string `yaml:"address"`
		Weight  int    `yaml:"weight,omitempty"`
	}

	w := u.Weight
	if w <= 0 {
		w = 1
	}

	return out{Name: u.Name, Address: u.Address, Weight: w}, nil
}

// UnmarshalYAML implements custom YAML unmarshaling for UpstreamConfig.
// It derives Type from Address if omitted and normalizes weight.
func (u *UpstreamConfig) UnmarshalYAML(unmarshal func(any) error) error {
	type in struct {
		Name    string `yaml:"name"`
		Address string `yaml:"address"`
		Type    string `yaml:"type,omitempty"`
		Weight  int    `yaml:"weight,omitempty"`
	}

	var tmp in
	if err := unmarshal(&tmp); err != nil {
		return err
	}

	u.Name = strings.TrimSpace(tmp.Name)

	u.Address = strings.TrimSpace(tmp.Address)
	if tmp.Weight <= 0 {
		u.Weight = 1
	} else {
		u.Weight = tmp.Weight
	}

	if tmp.Type != "" {
		u.Type = tmp.Type
	} else {
		u.Type = detectType(u.Address)
	}

	return nil
}

// Rule defines a DNS routing rule for internal use.
type Rule struct {
	Pattern string
	Via     string
	PinTTL  bool
}

// RuleGroup defines a group of related DNS rules.
type RuleGroup struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Via         string   `yaml:"via"`
	Patterns    []string `yaml:"patterns"`
	PinTTL      bool     `yaml:"pin_ttl,omitempty"`
}

// HistoryConfig defines query history settings.
type HistoryConfig struct {
	Enabled       bool `yaml:"enabled,omitempty"`
	MaxEntries    int  `yaml:"max_entries,omitempty"`
	Retention     int  `yaml:"retention_hours,omitempty"`
	BufferSize    int  `yaml:"buffer_size,omitempty"`
	FlushInterval int  `yaml:"flush_interval_ms,omitempty"`
	Compression   bool `yaml:"compression,omitempty"`
}

// LogConfig defines logging configuration (simplified - only level used).
type LogConfig struct {
	Level string `yaml:"level,omitempty"`
}

// CacheConfig defines DNS cache settings (simplified - only enabled used).
type CacheConfig struct {
	Enabled    bool `yaml:"enabled,omitempty"`
	MaxEntries int  `yaml:"max_entries,omitempty"`
	MaxSizeMB  int  `yaml:"max_size_mb,omitempty"`
	// MinTTLSeconds bounds the minimum TTL stored in cache (default 60s)
	MinTTLSeconds int `yaml:"min_ttl_seconds,omitempty"`
	// MaxTTLSeconds bounds the maximum TTL stored in cache (default 3600s)
	MaxTTLSeconds int `yaml:"max_ttl_seconds,omitempty"`
	// ServeStale enables serve-stale with background refresh (default true)
	ServeStale bool `yaml:"serve_stale,omitempty"`
}

// HTTPConfig defines HTTP admin server settings.
type HTTPConfig struct {
	Enabled        bool          `yaml:"enabled,omitempty"`
	Listen         string        `yaml:"listen,omitempty"`
	ReadTimeout    time.Duration `yaml:"read_timeout,omitempty"`
	WriteTimeout   time.Duration `yaml:"write_timeout,omitempty"`
	IdleTimeout    time.Duration `yaml:"idle_timeout,omitempty"`
	MaxHeaderBytes int           `yaml:"max_header_bytes,omitempty"`
}

// UpdateConfig defines automatic update settings.
type UpdateConfig struct {
	Enabled           bool `json:"enabled"            yaml:"enabled"`
	IncludePrerelease bool `json:"include_prerelease" yaml:"include_prerelease,omitempty"`
}

// LocalZonesConfig is removed - Local DNS is now fully auto-detected

// Config is the main application configuration.
type Config struct {
	AppName    string           `yaml:"app_name,omitempty"`
	Listen     ListenConfig     `yaml:"listen"`
	Upstreams  []UpstreamConfig `yaml:"upstreams"`
	RuleGroups []RuleGroup      `yaml:"rule_groups"`
	History    HistoryConfig    `yaml:"history,omitempty"`
	Log        LogConfig        `yaml:"log,omitempty"`
	Cache      CacheConfig      `yaml:"cache,omitempty"`
	HTTP       HTTPConfig       `yaml:"http,omitempty"`
	Hosts      []HostOverride   `yaml:"hosts,omitempty"`
	Update     UpdateConfig     `yaml:"update,omitempty"`
	// LocalZones removed - Local DNS is now fully auto-detected
	Path string `yaml:"-"`
}

// global mutex to serialize YAML writes.
var saveMu sync.Mutex //nolint:gochecknoglobals // global mutex for config writes

// HostOverride is a static host mapping (supports wildcard patterns like *.example.com).
type HostOverride struct {
	Pattern string   `json:"pattern"        yaml:"pattern"`
	A       []string `json:"a,omitempty"    yaml:"a,omitempty"`
	AAAA    []string `json:"aaaa,omitempty" yaml:"aaaa,omitempty"`
	TTL     uint32   `json:"ttl,omitempty"  yaml:"ttl,omitempty"`
}

func (c *Config) GetMinMarkTTL(ttl uint32) time.Duration {
	minTTL := 60
	if c.Cache.MinTTLSeconds > 0 {
		minTTL = c.Cache.MinTTLSeconds
	}

	maxTTL := 3600
	if c.Cache.MaxTTLSeconds > 0 {
		maxTTL = c.Cache.MaxTTLSeconds
	}

	v := max(minTTL, min(int(ttl), maxTTL))

	return time.Duration(v) * time.Second
}

// GetAllRules returns all rules from all rule groups with their via interface.
func (c *Config) GetAllRules() []Rule {
	var allRules []Rule

	// Add rules from groups
	for _, group := range c.RuleGroups {
		for _, pattern := range group.Patterns {
			rule := Rule{
				Pattern: pattern,
				Via:     group.Via,
				PinTTL:  group.PinTTL,
			}
			allRules = append(allRules, rule)
		}
	}

	return allRules
}

// GetEnabledUpstreams returns all upstream servers.
func (c *Config) GetEnabledUpstreams() []UpstreamConfig {
	return c.Upstreams
}

// GetUpstreamAddresses returns upstream addresses in legacy format for compatibility.
func (c *Config) GetUpstreamAddresses() []string {
	upstreams := c.GetEnabledUpstreams()
	addresses := make([]string, 0, len(upstreams))

	for _, upstream := range upstreams {
		addresses = append(addresses, upstream.Type+":"+upstream.Address)
	}

	return addresses
}

// GetUpstreamsByWeight returns upstreams sorted by weight (desc), default weight=1.
func (c *Config) GetUpstreamsByWeight() []UpstreamConfig {
	ups := make([]UpstreamConfig, len(c.Upstreams))
	copy(ups, c.Upstreams)

	for i := range ups {
		if ups[i].Weight <= 0 {
			ups[i].Weight = 1
		}
	}

	slices.SortFunc(ups, func(a, b UpstreamConfig) int {
		if a.Weight == b.Weight {
			return 0
		}

		if a.Weight > b.Weight {
			return -1
		}

		return 1
	})

	return ups
}

// GetRuleGroups returns all rule groups.
func (c *Config) GetRuleGroups() []RuleGroup {
	return c.RuleGroups
}

func Load(path string) (*Config, error) { //nolint:cyclop,funlen
	b, err := os.ReadFile(path) //nolint:gosec // config file path is validated
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}

	cfg.Path = path

	// Set defaults
	if cfg.AppName == "" {
		cfg.AppName = "outway"
	}

	if cfg.Listen.UDP == "" {
		cfg.Listen.UDP = ":53"
	}

	if cfg.Listen.TCP == "" {
		cfg.Listen.TCP = ":53"
	}

	// Set default upstreams if none configured
	if len(cfg.Upstreams) == 0 {
		cfg.Upstreams = []UpstreamConfig{
			{Name: "Cloudflare", Address: "1.1.1.1:53", Type: "udp"},
			{Name: "Google", Address: "8.8.8.8:53", Type: "udp"},
		}
	}

	// Ensure default weight and detect type if missing
	for i := range cfg.Upstreams {
		if cfg.Upstreams[i].Weight <= 0 {
			cfg.Upstreams[i].Weight = 1
		}

		if cfg.Upstreams[i].Type == "" {
			cfg.Upstreams[i].Type = detectType(cfg.Upstreams[i].Address)
		}
	}

	// Set default history settings
	if cfg.History.MaxEntries <= 0 {
		cfg.History.MaxEntries = 1000
	}

	if !cfg.History.Enabled {
		cfg.History.Enabled = true
	}

	// Set default log settings
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}

	// Set default cache settings
	if !cfg.Cache.Enabled {
		cfg.Cache.Enabled = true
	}

	if cfg.Cache.MaxEntries <= 0 && cfg.Cache.MaxSizeMB <= 0 {
		cfg.Cache.MaxEntries = 10000
	}

	// Set default cache TTL bounds and serve-stale behavior
	if cfg.Cache.MinTTLSeconds <= 0 {
		cfg.Cache.MinTTLSeconds = 60
	}

	if cfg.Cache.MaxTTLSeconds <= 0 {
		cfg.Cache.MaxTTLSeconds = 3600
	}

	if !cfg.Cache.ServeStale {
		cfg.Cache.ServeStale = true
	}

	// normalize invariants once
	if cfg.Cache.MaxTTLSeconds < cfg.Cache.MinTTLSeconds {
		cfg.Cache.MaxTTLSeconds = cfg.Cache.MinTTLSeconds
	}

	// Set default values for rule groups
	for i := range cfg.RuleGroups {
		// pin_ttl defaults to true (since it's omitempty, we need to check if it was explicitly set to false)
		// For now, we'll always set it to true as default
		cfg.RuleGroups[i].PinTTL = true
	}

	// Set default HTTP settings
	if cfg.HTTP.Listen == "" {
		cfg.HTTP.Listen = "127.0.0.1:47823"
	}

	if cfg.HTTP.ReadTimeout == 0 {
		cfg.HTTP.ReadTimeout = defaultHTTPReadTimeout
	}

	if cfg.HTTP.WriteTimeout == 0 {
		cfg.HTTP.WriteTimeout = defaultHTTPWriteTimeout
	}

	if cfg.HTTP.IdleTimeout == 0 {
		cfg.HTTP.IdleTimeout = defaultHTTPIdleTimeout
	}

	if cfg.HTTP.MaxHeaderBytes == 0 {
		cfg.HTTP.MaxHeaderBytes = defaultMaxHeaderBytes
	}

	if !cfg.HTTP.Enabled {
		cfg.HTTP.Enabled = true
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the configuration back to the original file path.
func (c *Config) Save() error {
	saveMu.Lock()
	defer saveMu.Unlock()

	if c.Path == "" {
		return errConfigPathEmpty
	}

	out, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(c.Path, out, defaultFilePerm)
}

func (c *Config) Validate() error { //nolint:gocognit,cyclop,funlen
	if c.Listen.UDP == "" || c.Listen.TCP == "" {
		return errListenUDPTCPMustBeSet
	}

	if err := validateAddr(c.Listen.UDP); err != nil {
		return fmt.Errorf("invalid listen.udp: %w", err)
	}

	if err := validateAddr(c.Listen.TCP); err != nil {
		return fmt.Errorf("invalid listen.tcp: %w", err)
	}

	if len(c.Upstreams) == 0 {
		return errAtLeastOneUpstreamRequired
	}

	for _, u := range c.Upstreams {
		if u.Name == "" {
			return errUpstreamNameCannotBeEmpty
		}

		// Cache limits sanity
		if c.Cache.Enabled {
			if c.Cache.MaxEntries < 0 || c.Cache.MaxSizeMB < 0 {
				return errCacheLimitsMustBeNonNegative
			}

			// TTL bounds sanity
			if c.Cache.MinTTLSeconds < 0 || c.Cache.MaxTTLSeconds < 0 {
				return errCacheTTLBoundsMustBeNonNeg
			}

			if c.Cache.MaxTTLSeconds > 0 && c.Cache.MinTTLSeconds > c.Cache.MaxTTLSeconds {
				return errCacheMinTTLGreaterThanMax
			}
		}

		if u.Address == "" {
			return fmt.Errorf("upstream '%s' %w", u.Name, errUpstreamAddressCannotBeEmpty)
		}
		// Type is optional and derived from URL; do not enforce here
		if u.Weight < 0 {
			return fmt.Errorf("upstream '%s' %w %d", u.Name, errUpstreamInvalidWeight, u.Weight)
		}
	}

	// Validate rule groups (optional)
	// Rule groups are optional - if present, they must be valid
	//nolint:nestif
	if len(c.RuleGroups) > 0 {
		groupNames := map[string]struct{}{}
		seen := map[string]struct{}{}

		for _, group := range c.RuleGroups {
			if group.Name == "" {
				return errRuleGroupNameCannotBeEmpty
			}

			if _, ok := groupNames[group.Name]; ok {
				return fmt.Errorf("%w: %s", errDuplicateRuleGroupName, group.Name)
			}

			groupNames[group.Name] = struct{}{}

			if len(group.Patterns) == 0 {
				return fmt.Errorf("rule group '%s': %w", group.Name, errRuleGroupMustHavePattern)
			}

			if group.Via == "" {
				return fmt.Errorf("rule group '%s': %w", group.Name, errRuleGroupRequiresViaInterface)
			}

			// Validate patterns within the group
			for _, pattern := range group.Patterns {
				if pattern == "" {
					return fmt.Errorf("rule group '%s': %w", group.Name, errRuleGroupContainsEmptyPattern)
				}

				if _, ok := seen[pattern]; ok {
					return fmt.Errorf("%w: %s", errDuplicateRulePattern, pattern)
				}

				seen[pattern] = struct{}{}
			}
		}
	}

	return nil
}

func validateAddr(addr string) error {
	if !strings.HasPrefix(addr, ":") && !strings.Contains(addr, ":") {
		return errAddressMustBeHostPort
	}

	_, _, err := net.SplitHostPort(addr)

	return err
}
