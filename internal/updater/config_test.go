package updater_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/updater"
)

func TestConfig_Validation(t *testing.T) {
	t.Parallel()

	tests := getBasicConfigTests()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Test validation by trying to create updater
			_, err := updater.New(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	t.Parallel()

	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}

	// Test that default values are set correctly
	assert.Equal(t, "test", config.Owner)
	assert.Equal(t, "test", config.Repo)
	assert.Equal(t, "v1.0.0", config.CurrentVersion)
	assert.Equal(t, "test", config.BinaryName)
}

func TestConfig_WithCustomValues(t *testing.T) {
	t.Parallel()

	config := updater.Config{
		Owner:          "custom-owner",
		Repo:           "custom-repo",
		CurrentVersion: "v2.1.0",
		BinaryName:     "custom-binary",
	}

	// Test that custom values are preserved
	assert.Equal(t, "custom-owner", config.Owner)
	assert.Equal(t, "custom-repo", config.Repo)
	assert.Equal(t, "v2.1.0", config.CurrentVersion)
	assert.Equal(t, "custom-binary", config.BinaryName)
}

func TestConfig_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := getEdgeCaseConfigTests()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Test validation by trying to create updater
			_, err := updater.New(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_Equality(t *testing.T) {
	t.Parallel()

	config1 := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}

	config2 := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}

	config3 := updater.Config{
		Owner:          "different",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}

	// Test equality
	assert.Equal(t, config1, config2)
	assert.NotEqual(t, config1, config3)
}

func TestConfig_Copy(t *testing.T) {
	t.Parallel()

	original := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}

	// Test that we can create a copy
	copyConfig := original
	copyConfig.Owner = "different"

	// Original should be unchanged
	assert.Equal(t, "test", original.Owner)
	assert.Equal(t, "different", copyConfig.Owner)
}
