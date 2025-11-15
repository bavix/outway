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

const (
	// MaxDNSNameLength is the maximum length of a DNS name (RFC 1035).
	MaxDNSNameLength = 253
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

	// HostOverride validation errors.
	errHostPatternEmpty             = errors.New("host pattern cannot be empty")
	errHostPatternTooLong           = errors.New("host pattern too long (max 253 characters)")
	errDomainLabelTooLongOrEmpty    = errors.New("invalid domain pattern: label too long or empty")
	errDomainLabelCannotStartHyphen = errors.New("invalid domain pattern: label cannot start with hyphen")
	errDomainLabelCannotEndHyphen   = errors.New("invalid domain pattern: label cannot end with hyphen")
	errHostOverrideNoRecords        = errors.New("host override must have at least one A or AAAA record")
	errARecordEmpty                 = errors.New("a record cannot be empty")
	errARecordInvalidIPv4           = errors.New("invalid IPv4 address in a record")
	errARecordNotIPv4               = errors.New("a record must be IPv4 address")
	errAAAARecordEmpty              = errors.New("aaaa record cannot be empty")
	errAAAARecordInvalidIPv6        = errors.New("invalid IPv6 address in aaaa record")
	errAAAARecordNotIPv6            = errors.New("aaaa record must be IPv6 address")
	errTTLTooLarge                  = errors.New("ttl too large")
	errDomainInvalidCharacter       = errors.New("invalid domain pattern: invalid character in label")
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
	MaxRequestSize int64         `yaml:"max_request_size,omitempty"` // Max request body size in bytes (default 1MB)
}

// UpdateConfig defines automatic update settings.
type UpdateConfig struct {
	Enabled           bool `json:"enabled"            yaml:"enabled"`
	IncludePrerelease bool `json:"include_prerelease" yaml:"include_prerelease,omitempty"`
}

// LocalZonesConfig is removed - Local DNS is now fully auto-detected

// Config is the main application configuration.
type Config struct {
	AppName       string           `yaml:"app_name,omitempty"`
	Listen        ListenConfig     `yaml:"listen"`
	Upstreams     []UpstreamConfig `yaml:"upstreams"`
	RuleGroups    []RuleGroup      `yaml:"rule_groups"`
	History       HistoryConfig    `yaml:"history,omitempty"`
	Log           LogConfig        `yaml:"log,omitempty"`
	Cache         CacheConfig      `yaml:"cache,omitempty"`
	HTTP          HTTPConfig       `yaml:"http,omitempty"`
	Hosts         []HostOverride   `yaml:"hosts,omitempty"`
	Update        UpdateConfig     `yaml:"update,omitempty"`
	Users         []UserConfig     `yaml:"users,omitempty"`
	JWTSecret     string           `yaml:"jwt_secret,omitempty"`     // Base64 encoded JWT secret
	RefreshTokens []RefreshToken   `yaml:"refresh_tokens,omitempty"` // Persisted refresh tokens
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

// UserConfig represents a user configuration.
type UserConfig struct {
	Email    string `json:"email"    yaml:"email"`
	Password string `json:"password" yaml:"password"` // This is a hash, not plain text
	Role     string `json:"role"     yaml:"role"`     // "admin" for now
}

// RefreshToken represents a persisted refresh token.
type RefreshToken struct {
	Token     string    `json:"token"      yaml:"token"`
	UserEmail string    `json:"user_email" yaml:"user_email"`
	ExpiresAt time.Time `json:"expires_at" yaml:"expires_at"`
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}

// Validate validates a HostOverride entry.
//
//nolint:gocognit,cyclop,funlen // complex validation logic with multiple checks
func (h *HostOverride) Validate() error {
	// Validate pattern
	if h.Pattern == "" {
		return errHostPatternEmpty
	}

	// Pattern should be a valid domain name or wildcard pattern
	pattern := strings.TrimSpace(h.Pattern)
	if len(pattern) > MaxDNSNameLength {
		return errHostPatternTooLong
	}

	// Allow wildcard patterns like *.example.com
	pattern = strings.TrimPrefix(pattern, "*.")

	// Basic domain validation - allow letters, numbers, dots, hyphens
	// Must start and end with alphanumeric
	if pattern != "" {
		parts := strings.SplitSeq(pattern, ".")
		for part := range parts {
			if len(part) == 0 || len(part) > 63 {
				return errDomainLabelTooLongOrEmpty
			}
			// Each label should be alphanumeric with optional hyphens
			for i, r := range part {
				if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
					return fmt.Errorf("%w: '%c'", errDomainInvalidCharacter, r)
				}

				if i == 0 && r == '-' {
					return errDomainLabelCannotStartHyphen
				}

				if i == len(part)-1 && r == '-' {
					return errDomainLabelCannotEndHyphen
				}
			}
		}
	}

	// Validate A records (IPv4)
	for i, ip := range h.A {
		if ip == "" {
			return fmt.Errorf("%w (#%d)", errARecordEmpty, i+1)
		}

		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			return fmt.Errorf("%w (#%d: %s)", errARecordInvalidIPv4, i+1, ip)
		}

		if parsedIP.To4() == nil {
			return fmt.Errorf("%w (#%d: %s)", errARecordNotIPv4, i+1, ip)
		}
	}

	// Validate AAAA records (IPv6)
	for i, ip := range h.AAAA {
		if ip == "" {
			return fmt.Errorf("%w (#%d)", errAAAARecordEmpty, i+1)
		}

		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			return fmt.Errorf("%w (#%d: %s)", errAAAARecordInvalidIPv6, i+1, ip)
		}

		if parsedIP.To4() != nil {
			return fmt.Errorf("%w (#%d: %s)", errAAAARecordNotIPv6, i+1, ip)
		}
	}

	// Validate TTL (optional, but if set should be reasonable)
	// DNS TTL is typically 0-2147483647 seconds (about 68 years)
	const maxTTL = 2147483647
	if h.TTL > maxTTL {
		return fmt.Errorf("%w (max %d seconds): %d", errTTLTooLarge, maxTTL, h.TTL)
	}

	// At least one record type should be present
	if len(h.A) == 0 && len(h.AAAA) == 0 {
		return errHostOverrideNoRecords
	}

	return nil
}

// SafeConfig represents a configuration without sensitive data for API responses.
type SafeConfig struct {
	AppName    string           `json:"app_name,omitempty"`
	Listen     ListenConfig     `json:"listen"`
	Upstreams  []UpstreamConfig `json:"upstreams"`
	RuleGroups []RuleGroup      `json:"rule_groups"`
	History    HistoryConfig    `json:"history,omitzero"`
	Log        LogConfig        `json:"log,omitzero"`
	Cache      CacheConfig      `json:"cache,omitzero"`
	HTTP       HTTPConfig       `json:"http,omitzero"`
	Hosts      []HostOverride   `json:"hosts,omitempty"`
	Update     UpdateConfig     `json:"update,omitzero"`
	Users      []UserConfig     `json:"users,omitempty"`
}

// ToSafeConfig converts Config to SafeConfig (without sensitive data).
func (c *Config) ToSafeConfig() SafeConfig {
	return SafeConfig{
		AppName:    c.AppName,
		Listen:     c.Listen,
		Upstreams:  c.Upstreams,
		RuleGroups: c.RuleGroups,
		History:    c.History,
		Log:        c.Log,
		Cache:      c.Cache,
		HTTP:       c.HTTP,
		Hosts:      c.Hosts,
		Update:     c.Update,
		Users:      c.Users,
	}
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
		// If address already contains scheme (://), use it as-is
		// Otherwise, prepend type: prefix for legacy format
		if strings.Contains(upstream.Address, "://") {
			addresses = append(addresses, upstream.Address)
		} else {
			addresses = append(addresses, upstream.Type+":"+upstream.Address)
		}
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
		cfg.History.MaxEntries = 500 // Reduced from 1000 to prevent OOM
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
		return fmt.Errorf("%w: config path is empty", errConfigPathEmpty)
	}

	out, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}

	if err := os.WriteFile(c.Path, out, defaultFilePerm); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", c.Path, err)
	}

	return nil
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
