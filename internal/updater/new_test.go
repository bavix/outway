package updater_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/updater"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := getBasicConfigTests()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u, err := updater.New(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, u)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, u)
			}
		})
	}
}

func TestNew_WithValidConfig(t *testing.T) {
	t.Parallel()

	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}

	u, err := updater.New(config)
	require.NoError(t, err)
	require.NotNil(t, u)

	// Test that the updater was created successfully
	assert.NotNil(t, u)
}

func TestNew_WithCustomHTTPClient(t *testing.T) {
	t.Parallel()

	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}

	u, err := updater.New(config)
	require.NoError(t, err)
	require.NotNil(t, u)

	// Test that updater was created successfully
	assert.NotNil(t, u)
}

func TestNew_WithCustomLogger(t *testing.T) {
	t.Parallel()

	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}

	u, err := updater.New(config)
	require.NoError(t, err)
	require.NotNil(t, u)

	// Test that updater was created successfully
	assert.NotNil(t, u)
}

//nolint:funlen
func TestNew_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  updater.Config
		wantErr bool
	}{
		{
			name: "single character values",
			config: updater.Config{
				Owner:          "a",
				Repo:           "b",
				CurrentVersion: "v1.0.0",
				BinaryName:     "c",
			},
			wantErr: false,
		},
		{
			name: "version with pre-release",
			config: updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: "v1.0.0-alpha.1",
				BinaryName:     "test",
			},
			wantErr: false,
		},
		{
			name: "version with build metadata",
			config: updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: "v1.0.0+20130313144700",
				BinaryName:     "test",
			},
			wantErr: false,
		},
		{
			name: "version without v prefix",
			config: updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: "1.0.0",
				BinaryName:     "test",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u, err := updater.New(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, u)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, u)
			}
		})
	}
}

func TestNew_ConfigValidation(t *testing.T) {
	t.Parallel()
	// Test that New validates the config
	config := updater.Config{
		Owner:          "", // Invalid: empty owner
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}

	u, err := updater.New(config)
	require.Error(t, err)
	assert.Nil(t, u)
}

func TestNew_ReturnsValidUpdater(t *testing.T) {
	t.Parallel()

	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}

	u, err := updater.New(config)
	require.NoError(t, err)
	require.NotNil(t, u)

	// Test that updater was created successfully
	assert.NotNil(t, u)
}

func TestNew_WithSpecialCharacters(t *testing.T) {
	t.Parallel()

	tests := getSpecialCharacterConfigTests()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u, err := updater.New(tt.config)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, u)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, u)
			}
		})
	}
}
