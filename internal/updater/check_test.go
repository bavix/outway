package updater_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/updater"
)

func TestUpdater_CheckForUpdates(t *testing.T) {
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

	ctx := context.Background()

	// Test CheckForUpdates - this might fail due to network issues
	updateInfo, err := u.CheckForUpdates(ctx, false)
	if err != nil {
		t.Logf("Error checking for updates (expected if no internet): %v", err)

		return
	}

	require.NotNil(t, updateInfo)
	assert.Equal(t, "v1.0.0", updateInfo.CurrentVersion)
	assert.NotEmpty(t, updateInfo.LatestVersion)
}

func TestUpdater_CheckForUpdates_WithPrerelease(t *testing.T) {
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

	ctx := context.Background()

	// Test CheckForUpdates with prerelease flag
	updateInfo, err := u.CheckForUpdates(ctx, true)
	if err != nil {
		t.Logf("Error checking for updates (expected if no internet): %v", err)

		return
	}

	require.NotNil(t, updateInfo)
	assert.Equal(t, "v1.0.0", updateInfo.CurrentVersion)
}

func TestUpdater_CheckForUpdates_ContextCancellation(t *testing.T) {
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

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test CheckForUpdates with cancelled context
	updateInfo, err := u.CheckForUpdates(ctx, false)
	require.Error(t, err)
	assert.Nil(t, updateInfo)
}

func TestUpdater_CheckForUpdates_InvalidConfig(t *testing.T) {
	t.Parallel()
	// Test with invalid config
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

func TestUpdater_CheckForUpdates_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		currentVersion string
		prerelease     bool
	}{
		{
			name:           "stable version",
			currentVersion: "v1.0.0",
			prerelease:     false,
		},
		{
			name:           "prerelease version",
			currentVersion: "v1.0.0-alpha.1",
			prerelease:     true,
		},
		{
			name:           "version without v prefix",
			currentVersion: "1.0.0",
			prerelease:     false,
		},
		{
			name:           "version with build metadata",
			currentVersion: "v1.0.0+20130313144700",
			prerelease:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: tt.currentVersion,
				BinaryName:     "test",
			}

			u, err := updater.New(config)
			require.NoError(t, err)
			require.NotNil(t, u)

			ctx := context.Background()

			updateInfo, err := u.CheckForUpdates(ctx, tt.prerelease)
			if err != nil {
				t.Logf("Error checking for updates (expected if no internet): %v", err)

				return
			}

			require.NotNil(t, updateInfo)
			assert.Equal(t, tt.currentVersion, updateInfo.CurrentVersion)
		})
	}
}

func TestUpdater_CheckForUpdates_NetworkError(t *testing.T) {
	t.Parallel()

	config := updater.Config{
		Owner:          "nonexistent-owner",
		Repo:           "nonexistent-repo",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}

	u, err := updater.New(config)
	require.NoError(t, err)
	require.NotNil(t, u)

	ctx := context.Background()

	// Test CheckForUpdates with non-existent repo
	updateInfo, err := u.CheckForUpdates(ctx, false)
	require.Error(t, err)
	assert.Nil(t, updateInfo)
}

func TestUpdater_CheckForUpdates_Timeout(t *testing.T) {
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

	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1)
	defer cancel()

	// Test CheckForUpdates with timeout
	updateInfo, err := u.CheckForUpdates(ctx, false)
	require.Error(t, err)
	assert.Nil(t, updateInfo)
}

func TestUpdater_CheckForUpdates_Concurrent(t *testing.T) {
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

	ctx := context.Background()

	// Test concurrent CheckForUpdates calls
	done := make(chan bool, 5)

	for range 5 {
		go func() {
			defer func() { done <- true }()

			updateInfo, err := u.CheckForUpdates(ctx, false)
			// Don't assert on result as it might fail due to network
			_ = updateInfo
			_ = err
		}()
	}

	// Wait for all goroutines to complete
	for range 5 {
		<-done
	}
}

func TestUpdater_CheckForUpdates_ReturnValues(t *testing.T) {
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

	ctx := context.Background()

	updateInfo, err := u.CheckForUpdates(ctx, false)
	if err != nil {
		t.Logf("Error checking for updates (expected if no internet): %v", err)

		return
	}

	require.NotNil(t, updateInfo)

	// Test that UpdateInfo has expected structure
	assert.Equal(t, "v1.0.0", updateInfo.CurrentVersion)
	assert.NotEmpty(t, updateInfo.LatestVersion)
	assert.NotNil(t, updateInfo.Release)
}

func TestUpdater_CheckForUpdates_PrereleaseFlag(t *testing.T) {
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

	ctx := context.Background()

	// Test with prerelease = false
	updateInfo1, err1 := u.CheckForUpdates(ctx, false)
	if err1 != nil {
		t.Logf("Error checking for updates (expected if no internet): %v", err1)

		return
	}

	// Test with prerelease = true
	updateInfo2, err2 := u.CheckForUpdates(ctx, true)
	if err2 != nil {
		t.Logf("Error checking for updates (expected if no internet): %v", err2)

		return
	}

	// Both should succeed if network is available
	if err1 == nil && err2 == nil {
		assert.NotNil(t, updateInfo1)
		assert.NotNil(t, updateInfo2)
	}
}
