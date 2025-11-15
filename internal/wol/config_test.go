package wol_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/wol"
)

func TestConfig_DefaultConfig(t *testing.T) {
	t.Parallel()

	config := wol.DefaultConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 9, config.DefaultPort)
	assert.Equal(t, 5*time.Second, config.DefaultTimeout)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 1*time.Second, config.RetryDelay)
	assert.True(t, config.Enabled)
}

func TestConfigManager_NewConfigManager(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	assert.NotNil(t, manager)
	// assert.NotNil(t, manager.config) // unexported field, cannot access
	assert.Equal(t, wol.DefaultConfig(), manager.GetConfig())
}

func TestConfigManager_GetConfig(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	config := manager.GetConfig()
	assert.NotNil(t, config)
	assert.Equal(t, wol.DefaultConfig(), config)

	// Verify it returns a copy
	config.DefaultPort = 999
	assert.NotEqual(t, 999, manager.GetConfig().DefaultPort)
}

func TestConfigManager_SetConfig(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Test valid config
	newConfig := &wol.Config{
		DefaultPort:    7,
		DefaultTimeout: 10 * time.Second,
		MaxRetries:     5,
		RetryDelay:     2 * time.Second,
		Enabled:        false,
	}

	err := manager.SetConfig(newConfig)
	require.NoError(t, err)

	config := manager.GetConfig()
	assert.Equal(t, newConfig, config)

	// Test nil config
	err = manager.SetConfig(nil)
	require.Error(t, err)

	// Test invalid config
	invalidConfig := &wol.Config{
		DefaultPort: 0, // Invalid port
	}

	err = manager.SetConfig(invalidConfig)
	require.Error(t, err)
}

func testConfigUpdate(t *testing.T, configManager *wol.ConfigManager) {
	t.Helper()
	// Test valid updates
	updates := map[string]any{
		"default_port":    7,
		"default_timeout": 10.0, // Test float64 conversion
		"max_retries":     5,
		"retry_delay":     2.0, // Test float64 conversion
		"enabled":         false,
	}

	err := configManager.UpdateConfig(updates)
	require.NoError(t, err)

	config := configManager.GetConfig()
	assert.Equal(t, 7, config.DefaultPort)
	assert.Equal(t, 10*time.Second, config.DefaultTimeout)
	assert.Equal(t, 5, config.MaxRetries)
	assert.Equal(t, 2*time.Second, config.RetryDelay)
	assert.False(t, config.Enabled)

	// Test invalid updates
	invalidUpdates := map[string]any{
		"default_port": "invalid",
	}

	err = configManager.UpdateConfig(invalidUpdates)
	require.Error(t, err)

	// Test unknown key
	unknownUpdates := map[string]any{
		"unknown_key": "value",
	}

	err = configManager.UpdateConfig(unknownUpdates)
	require.Error(t, err)
}

func TestConfigManager_UpdateConfig(t *testing.T) {
	t.Parallel()
	testConfigUpdate(t, wol.NewConfigManager())
}

func TestConfigManager_IsEnabled(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Default should be enabled
	assert.True(t, manager.IsEnabled())

	// Disable
	updates := map[string]any{
		"enabled": false,
	}
	err := manager.UpdateConfig(updates)
	require.NoError(t, err)
	assert.False(t, manager.IsEnabled())

	// Re-enable
	updates = map[string]any{
		"enabled": true,
	}
	err = manager.UpdateConfig(updates)
	require.NoError(t, err)
	assert.True(t, manager.IsEnabled())
}

func TestConfigManager_GetDefaultPort(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Default port
	assert.Equal(t, 9, manager.GetDefaultPort())

	// Update port
	updates := map[string]any{
		"default_port": 7,
	}
	err := manager.UpdateConfig(updates)
	require.NoError(t, err)
	assert.Equal(t, 7, manager.GetDefaultPort())
}

func TestConfigManager_GetDefaultTimeout(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Default timeout
	assert.Equal(t, 5*time.Second, manager.GetDefaultTimeout())

	// Update timeout
	updates := map[string]any{
		"default_timeout": 10.0,
	}
	err := manager.UpdateConfig(updates)
	require.NoError(t, err)
	assert.Equal(t, 10*time.Second, manager.GetDefaultTimeout())
}

func TestConfigManager_GetMaxRetries(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Default retries
	assert.Equal(t, 3, manager.GetMaxRetries())

	// Update retries
	updates := map[string]any{
		"max_retries": 5,
	}
	err := manager.UpdateConfig(updates)
	require.NoError(t, err)
	assert.Equal(t, 5, manager.GetMaxRetries())
}

func TestConfigManager_GetRetryDelay(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Default delay
	assert.Equal(t, 1*time.Second, manager.GetRetryDelay())

	// Update delay
	updates := map[string]any{
		"retry_delay": 2.0,
	}
	err := manager.UpdateConfig(updates)
	require.NoError(t, err)
	assert.Equal(t, 2*time.Second, manager.GetRetryDelay())
}

func TestConfigManager_ToJSON(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Test JSON serialization
	jsonData, err := manager.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Verify it's valid JSON
	var config wol.Config

	err = json.Unmarshal(jsonData, &config)
	require.NoError(t, err)
	assert.Equal(t, manager.GetConfig(), &config)
}

func TestConfigManager_FromJSON(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Test JSON deserialization
	jsonData, err := manager.ToJSON()
	require.NoError(t, err)

	newManager := wol.NewConfigManager()
	err = newManager.FromJSON(jsonData)
	require.NoError(t, err)

	// Verify configs match
	assert.Equal(t, manager.GetConfig(), newManager.GetConfig())

	// Test invalid JSON
	err = newManager.FromJSON([]byte("invalid json"))
	require.Error(t, err)
}

func TestConfigManager_Reset(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Update config
	updates := map[string]any{
		"default_port": 7,
		"enabled":      false,
	}
	err := manager.UpdateConfig(updates)
	require.NoError(t, err)

	// Verify config changed
	config := manager.GetConfig()
	assert.Equal(t, 7, config.DefaultPort)
	assert.False(t, config.Enabled)

	// Reset config
	manager.Reset()

	// Verify config is back to defaults
	defaultConfig := manager.GetConfig()
	assert.Equal(t, wol.DefaultConfig(), defaultConfig)
}

func TestConfigManager_Clone(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Update config
	updates := map[string]any{
		"default_port": 7,
		"enabled":      false,
	}
	err := manager.UpdateConfig(updates)
	require.NoError(t, err)

	// Clone config
	clonedManager := manager.Clone()
	assert.NotNil(t, clonedManager)

	// Verify they have the same values
	assert.Equal(t, manager.GetConfig(), clonedManager.GetConfig())

	// Verify they are independent
	_ = manager.UpdateConfig(map[string]any{"default_port": 8})
	assert.NotEqual(t, manager.GetConfig().DefaultPort, clonedManager.GetConfig().DefaultPort)
}

func TestConfigManager_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Test concurrent access
	done := make(chan bool, 10)

	for i := range 10 {
		go func() {
			defer func() { done <- true }()

			// Read config
			config := manager.GetConfig()
			assert.NotNil(t, config)

			// Update config
			updates := map[string]any{
				"default_port": i,
			}
			_ = manager.UpdateConfig(updates)
		}()
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}

	// Verify final state is consistent
	config := manager.GetConfig()
	assert.NotNil(t, config)
	assert.GreaterOrEqual(t, config.DefaultPort, 0)
	assert.LessOrEqual(t, config.DefaultPort, 9)
}

//nolint:funlen
func TestConfigManager_Validation(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	tests := []struct {
		name     string
		config   *wol.Config
		hasError bool
	}{
		{
			name:     "valid config",
			config:   wol.DefaultConfig(),
			hasError: false,
		},
		{
			name: "invalid port too low",
			config: &wol.Config{
				DefaultPort: 0,
			},
			hasError: true,
		},
		{
			name: "invalid port too high",
			config: &wol.Config{
				DefaultPort: 65536,
			},
			hasError: true,
		},
		{
			name: "invalid timeout",
			config: &wol.Config{
				DefaultTimeout: -1 * time.Second,
			},
			hasError: true,
		},
		{
			name: "invalid max retries",
			config: &wol.Config{
				MaxRetries: -1,
			},
			hasError: true,
		},
		{
			name: "invalid retry delay",
			config: &wol.Config{
				RetryDelay: -1 * time.Second,
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := manager.SetConfig(tt.config)
			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

//nolint:funlen
func TestConfigManager_UpdateConfigTypes(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	tests := []struct {
		name     string
		updates  map[string]any
		hasError bool
	}{
		{
			name: "valid int types",
			updates: map[string]any{
				"default_port": 7,
				"max_retries":  5,
			},
			hasError: false,
		},
		{
			name: "valid float64 types",
			updates: map[string]any{
				"default_timeout": 10.0,
				"retry_delay":     2.0,
			},
			hasError: false,
		},
		{
			name: "valid bool type",
			updates: map[string]any{
				"enabled": false,
			},
			hasError: false,
		},
		{
			name: "invalid string type",
			updates: map[string]any{
				"default_port": "invalid",
			},
			hasError: true,
		},
		{
			name: "invalid type for timeout",
			updates: map[string]any{
				"default_timeout": "invalid",
			},
			hasError: true,
		},
		{
			name: "invalid type for retries",
			updates: map[string]any{
				"max_retries": "invalid",
			},
			hasError: true,
		},
		{
			name: "invalid type for delay",
			updates: map[string]any{
				"retry_delay": "invalid",
			},
			hasError: true,
		},
		{
			name: "invalid type for enabled",
			updates: map[string]any{
				"enabled": "invalid",
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := manager.UpdateConfig(tt.updates)
			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfigManager_EdgeCases(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Test empty updates
	err := manager.UpdateConfig(map[string]any{})
	require.NoError(t, err)

	// Test nil updates
	err = manager.UpdateConfig(nil)
	require.NoError(t, err)

	// Test partial updates
	updates := map[string]any{
		"default_port": 7,
	}
	err = manager.UpdateConfig(updates)
	require.NoError(t, err)

	config := manager.GetConfig()
	assert.Equal(t, 7, config.DefaultPort)
	assert.Equal(t, wol.DefaultConfig().DefaultTimeout, config.DefaultTimeout) // Should remain unchanged
}

func TestConfigManager_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Update config
	updates := map[string]any{
		"default_port":    7,
		"default_timeout": 10.0,
		"max_retries":     5,
		"retry_delay":     2.0,
		"enabled":         false,
	}
	err := manager.UpdateConfig(updates)
	require.NoError(t, err)

	// Convert to JSON
	jsonData, err := manager.ToJSON()
	require.NoError(t, err)

	// Create new manager and load from JSON
	newManager := wol.NewConfigManager()
	err = newManager.FromJSON(jsonData)
	require.NoError(t, err)

	// Verify configs match
	assert.Equal(t, manager.GetConfig(), newManager.GetConfig())
}

func TestConfigManager_ResetAfterUpdate(t *testing.T) {
	t.Parallel()

	manager := wol.NewConfigManager()

	// Update config
	updates := map[string]any{
		"default_port": 7,
		"enabled":      false,
	}
	err := manager.UpdateConfig(updates)
	require.NoError(t, err)

	// Verify update
	config := manager.GetConfig()
	assert.Equal(t, 7, config.DefaultPort)
	assert.False(t, config.Enabled)

	// Reset
	manager.Reset()

	// Verify reset
	config = manager.GetConfig()
	assert.Equal(t, wol.DefaultConfig(), config)
	assert.Equal(t, 9, config.DefaultPort)
	assert.True(t, config.Enabled)
}
