package wol

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bavix/outway/internal/errors"
)

const (
	defaultWOLPort    = 9
	defaultTimeout    = 5 * time.Second
	defaultMaxRetries = 3
	defaultRetryDelay = 1 * time.Second
	ipv4AddressLength = 4
	broadcastMask     = 0xff
	macAddressLength  = 12
	macBytesLength    = 6
	magicPacketSize   = 102
	scanTimeout       = 500 * time.Millisecond
	maxConcurrency    = 100
	maxIPsPerBatch    = 20
	minNmapParts      = 5
	minArpParts       = 2
	arpMatchesCount   = 3
)

// Config represents Wake-on-LAN configuration.
type Config struct {
	DefaultPort    int           `json:"default_port"    yaml:"default_port"`
	DefaultTimeout time.Duration `json:"default_timeout" yaml:"default_timeout"`
	MaxRetries     int           `json:"max_retries"     yaml:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay"     yaml:"retry_delay"`
	Enabled        bool          `json:"enabled"         yaml:"enabled"`
}

// DefaultConfig returns the default Wake-on-LAN configuration.
func DefaultConfig() *Config {
	return &Config{
		DefaultPort:    defaultWOLPort,    // Standard WOL port
		DefaultTimeout: defaultTimeout,    // 5 second timeout
		MaxRetries:     defaultMaxRetries, // 3 retries
		RetryDelay:     defaultRetryDelay, // 1 second between retries
		Enabled:        true,              // Enabled by default
	}
}

// ConfigManager manages Wake-on-LAN configuration.
type ConfigManager struct {
	config *Config
	mu     sync.RWMutex
}

// NewConfigManager creates a new configuration manager.
func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		config: DefaultConfig(),
	}
}

// GetConfig returns the current configuration.
func (cm *ConfigManager) GetConfig() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Return a copy to prevent external modifications
	configCopy := *cm.config

	return &configCopy
}

// SetConfig updates the configuration.
func (cm *ConfigManager) SetConfig(config *Config) error {
	if config == nil {
		return errors.ErrConfigCannotBeNil
	}

	// Validate configuration
	if err := cm.validateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.config = config

	return nil
}

// UpdateConfig updates specific configuration fields.
func (cm *ConfigManager) UpdateConfig(updates map[string]any) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Create a copy of current config
	newConfig := *cm.config

	// Apply updates
	for key, value := range updates {
		if err := cm.updateConfigField(&newConfig, key, value); err != nil {
			return err
		}
	}

	// Validate updated configuration
	if err := cm.validateConfig(&newConfig); err != nil {
		return fmt.Errorf("invalid updated configuration: %w", err)
	}

	cm.config = &newConfig

	return nil
}

// IsEnabled returns whether Wake-on-LAN is enabled.
func (cm *ConfigManager) IsEnabled() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.config.Enabled
}

// GetDefaultPort returns the default port.
func (cm *ConfigManager) GetDefaultPort() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.config.DefaultPort
}

// GetDefaultTimeout returns the default timeout.
func (cm *ConfigManager) GetDefaultTimeout() time.Duration {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.config.DefaultTimeout
}

// GetMaxRetries returns the maximum number of retries.
func (cm *ConfigManager) GetMaxRetries() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.config.MaxRetries
}

// GetRetryDelay returns the delay between retries.
func (cm *ConfigManager) GetRetryDelay() time.Duration {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.config.RetryDelay
}

// ToJSON converts the configuration to JSON.
func (cm *ConfigManager) ToJSON() ([]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return json.Marshal(cm.config)
}

// FromJSON loads configuration from JSON.
func (cm *ConfigManager) FromJSON(data []byte) error {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cm.SetConfig(&config)
}

// Reset resets the configuration to defaults.
func (cm *ConfigManager) Reset() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.config = DefaultConfig()
}

// Clone creates a deep copy of the configuration.
func (cm *ConfigManager) Clone() *ConfigManager {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	configCopy := *cm.config

	return &ConfigManager{
		config: &configCopy,
	}
}

func (cm *ConfigManager) updateConfigField(config *Config, key string, value any) error {
	switch key {
	case "default_port":
		return cm.updateDefaultPort(config, value)
	case "default_timeout":
		return cm.updateDefaultTimeout(config, value)
	case "max_retries":
		return cm.updateMaxRetries(config, value)
	case "retry_delay":
		return cm.updateRetryDelay(config, value)
	case "enabled":
		return cm.updateEnabled(config, value)
	default:
		return errors.ErrUnknownConfigurationKey(key)
	}
}

func (cm *ConfigManager) updateDefaultPort(config *Config, value any) error {
	if port, ok := value.(int); ok {
		config.DefaultPort = port
	} else if port, ok := value.(float64); ok {
		config.DefaultPort = int(port)
	} else {
		return errors.ErrInvalidTypeForDefaultPort(value)
	}

	return nil
}

func (cm *ConfigManager) updateDefaultTimeout(config *Config, value any) error {
	if timeout, ok := value.(time.Duration); ok {
		config.DefaultTimeout = timeout
	} else if timeout, ok := value.(float64); ok {
		config.DefaultTimeout = time.Duration(timeout) * time.Second
	} else if timeout, ok := value.(int); ok {
		config.DefaultTimeout = time.Duration(timeout) * time.Second
	} else {
		return errors.ErrInvalidTypeForDefaultTimeout(value)
	}

	return nil
}

func (cm *ConfigManager) updateMaxRetries(config *Config, value any) error {
	if retries, ok := value.(int); ok {
		config.MaxRetries = retries
	} else if retries, ok := value.(float64); ok {
		config.MaxRetries = int(retries)
	} else {
		return errors.ErrInvalidTypeForMaxRetries(value)
	}

	return nil
}

func (cm *ConfigManager) updateRetryDelay(config *Config, value any) error {
	if delay, ok := value.(time.Duration); ok {
		config.RetryDelay = delay
	} else if delay, ok := value.(float64); ok {
		config.RetryDelay = time.Duration(delay) * time.Second
	} else if delay, ok := value.(int); ok {
		config.RetryDelay = time.Duration(delay) * time.Second
	} else {
		return errors.ErrInvalidTypeForRetryDelay(value)
	}

	return nil
}

func (cm *ConfigManager) updateEnabled(config *Config, value any) error {
	if enabled, ok := value.(bool); ok {
		config.Enabled = enabled
	} else {
		return errors.ErrInvalidTypeForEnabled(value)
	}

	return nil
}

func (cm *ConfigManager) validateConfig(config *Config) error {
	if config.DefaultPort < 1 || config.DefaultPort > 65535 {
		return errors.ErrDefaultPortInvalid(config.DefaultPort)
	}

	if config.DefaultTimeout <= 0 {
		return errors.ErrDefaultTimeoutInvalid(config.DefaultTimeout)
	}

	if config.MaxRetries < 0 {
		return errors.ErrMaxRetriesInvalid(config.MaxRetries)
	}

	if config.RetryDelay < 0 {
		return errors.ErrRetryDelayInvalid(config.RetryDelay)
	}

	return nil
}
