package updater_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/updater"
)

//nolint:funlen
func TestUpdater_isNewerVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		current     string
		latest      string
		expected    bool
		description string
		skip        bool
	}{
		{
			name:        "same version",
			current:     "v1.0.0",
			latest:      "v1.0.0",
			expected:    false,
			description: "Same version should not be newer",
		},
		{
			name:        "current newer than latest",
			current:     "v2.0.0",
			latest:      "v1.0.0",
			expected:    true,
			description: "Current version newer than latest should be newer",
		},
		{
			name:        "latest newer than current",
			current:     "v1.0.0",
			latest:      "v2.0.0",
			expected:    false,
			description: "Latest version newer than current should not be newer (current is older)",
		},
		{
			name:        "patch version update",
			current:     "v1.0.0",
			latest:      "v1.0.1",
			expected:    false,
			description: "Patch version update should not be newer (current is older)",
		},
		{
			name:        "minor version update",
			current:     "v1.0.0",
			latest:      "v1.1.0",
			expected:    false,
			description: "Minor version update should not be newer (current is older)",
		},
		{
			name:        "major version update",
			current:     "v1.0.0",
			latest:      "v2.0.0",
			expected:    false,
			description: "Major version update should not be newer (current is older)",
		},
		{
			name:        "prerelease vs stable",
			current:     "v1.0.0-alpha.1",
			latest:      "v1.0.0",
			expected:    false,
			description: "Stable version should not be newer than prerelease (current is prerelease)",
		},
		{
			name:        "stable vs prerelease",
			current:     "v1.0.0",
			latest:      "v1.0.0-alpha.1",
			expected:    true,
			description: "Stable version should be newer than prerelease",
		},
		{
			name:        "prerelease vs prerelease",
			current:     "v1.0.0-alpha.1",
			latest:      "v1.0.0-alpha.2",
			expected:    false,
			description: "Newer prerelease should not be newer (current is older prerelease)",
		},
		{
			name:        "build metadata",
			current:     "v1.0.0+20130313144700",
			latest:      "v1.0.0+20130313144701",
			expected:    false,
			description: "Build metadata should not affect version comparison",
		},
		{
			name:        "version without v prefix",
			current:     "1.0.0",
			latest:      "2.0.0",
			expected:    false,
			description: "Version without v prefix should not be newer (current is older)",
		},
		{
			name:        "mixed v prefix",
			current:     "v1.0.0",
			latest:      "2.0.0",
			expected:    false,
			description: "Mixed v prefix should not be newer (current is older)",
		},
		{
			name:        "invalid current version",
			current:     "invalid",
			latest:      "v1.0.0",
			expected:    false,
			description: "Invalid current version should not be newer",
		},
		{
			name:        "invalid latest version",
			current:     "v1.0.0",
			latest:      "invalid",
			expected:    true,
			description: "Invalid latest version should be newer (current is valid)",
		},
		{
			name:        "both invalid versions",
			current:     "invalid",
			latest:      "invalid",
			expected:    false,
			description: "Both invalid versions should not be newer",
		},
		{
			name:        "empty current version",
			current:     "",
			latest:      "v1.0.0",
			expected:    false,
			description: "Empty current version should not be newer",
			skip:        true, // Skip because updater.New will fail
		},
		{
			name:        "empty latest version",
			current:     "v1.0.0",
			latest:      "",
			expected:    true,
			description: "Empty latest version should be newer (current is valid)",
		},
		{
			name:        "both empty versions",
			current:     "",
			latest:      "",
			expected:    false,
			description: "Both empty versions should not be newer",
			skip:        true, // Skip because updater.New will fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.skip {
				t.Skip("Skipping test due to updater.New failure")
			}

			// Test isNewerVersion through updater instance
			u, err := updater.New(updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: tt.current,
				BinaryName:     "test",
			})
			if err != nil {
				t.Errorf("Failed to create updater: %v", err)

				return
			}

			result := u.IsNewerVersion(tt.current, tt.latest)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestUpdater_isNewerVersion_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := getEdgeCaseVersionTests()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.skip {
				t.Skip("Skipping test due to updater.New failure")
			}

			// Test isNewerVersion through updater instance
			u, err := updater.New(updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: tt.current,
				BinaryName:     "test",
			})
			if err != nil {
				t.Errorf("Failed to create updater: %v", err)

				return
			}

			result := u.IsNewerVersion(tt.current, tt.latest)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func getEdgeCaseVersionTests() []struct {
	name        string
	current     string
	latest      string
	expected    bool
	description string
	skip        bool
} {
	return []struct {
		name        string
		current     string
		latest      string
		expected    bool
		description string
		skip        bool
	}{
		{
			name:        "zero versions",
			current:     "v0.0.0",
			latest:      "v0.0.0",
			expected:    false,
			description: "Zero versions should be equal",
		},
		{
			name:        "zero to one",
			current:     "v0.0.0",
			latest:      "v1.0.0",
			expected:    false,
			description: "Zero to one should not be newer (current is older)",
		},
		{
			name:        "very large version",
			current:     "v999.999.999",
			latest:      "v1000.0.0",
			expected:    false,
			description: "Very large version should not be newer (current is older)",
		},
		{
			name:        "version with many prerelease parts",
			current:     "v1.0.0-alpha.1.beta.2.rc.3",
			latest:      "v1.0.0-alpha.1.beta.2.rc.4",
			expected:    false,
			description: "Version with many prerelease parts should not be newer (current is older)",
		},
		{
			name:        "version with long build metadata",
			current:     "v1.0.0+20130313144700.123456789",
			latest:      "v1.0.0+20130313144700.123456790",
			expected:    false,
			description: "Version with long build metadata should not affect comparison",
		},
	}
}

func TestUpdater_isNewerVersion_Concurrent(t *testing.T) {
	t.Parallel()
	// Test concurrent calls to IsNewerVersion
	done := make(chan bool, 10)

	for range 10 {
		go func() {
			defer func() { done <- true }()

			u, err := updater.New(updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: "v1.0.0",
				BinaryName:     "test",
			})
			if err != nil {
				t.Errorf("Failed to create updater: %v", err)

				return
			}

			result := u.IsNewerVersion("v1.0.0", "v2.0.0")
			assert.False(t, result)
		}()
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}
}

func TestUpdater_isNewerVersion_Performance(t *testing.T) {
	t.Parallel()
	// Test performance with many calls
	u, err := updater.New(updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	})
	require.NoError(t, err)

	for range 1000 {
		result := u.IsNewerVersion("v1.0.0", "v2.0.0")
		assert.False(t, result)
	}
}

func TestUpdater_isNewerVersion_RealWorldExamples(t *testing.T) {
	t.Parallel()

	tests := getRealWorldVersionTests()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.skip {
				t.Skip("Skipping test due to updater.New failure")
			}

			// Test isNewerVersion through updater instance
			u, err := updater.New(updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: tt.current,
				BinaryName:     "test",
			})
			if err != nil {
				t.Errorf("Failed to create updater: %v", err)

				return
			}

			result := u.IsNewerVersion(tt.current, tt.latest)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func getRealWorldVersionTests() []struct {
	name        string
	current     string
	latest      string
	expected    bool
	description string
	skip        bool
} {
	return []struct {
		name        string
		current     string
		latest      string
		expected    bool
		description string
		skip        bool
	}{
		{
			name:        "Go version style",
			current:     "v1.21.0",
			latest:      "v1.21.1",
			expected:    false,
			description: "Go version style should not be newer (current is older)",
		},
		{
			name:        "Docker version style",
			current:     "v24.0.0",
			latest:      "v24.0.1",
			expected:    false,
			description: "Docker version style should not be newer (current is older)",
		},
		{
			name:        "Kubernetes version style",
			current:     "v1.28.0",
			latest:      "v1.28.1",
			expected:    false,
			description: "Kubernetes version style should not be newer (current is older)",
		},
		{
			name:        "Node.js version style",
			current:     "v18.17.0",
			latest:      "v18.17.1",
			expected:    false,
			description: "Node.js version style should not be newer (current is older)",
		},
		{
			name:        "Python version style",
			current:     "v3.11.0",
			latest:      "v3.11.1",
			expected:    false,
			description: "Python version style should not be newer (current is older)",
		},
	}
}
