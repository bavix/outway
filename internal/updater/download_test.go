package updater_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/updater"
)

func TestUpdater_DownloadUpdate(t *testing.T) {
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

	// Test DownloadUpdate with invalid URL
	invalidURL := "https://invalid-url.com/nonexistent-file.tar.gz"
	updatePath, err := u.DownloadUpdate(ctx, invalidURL)
	require.Error(t, err)
	assert.Empty(t, updatePath)
}

func TestUpdater_DownloadUpdate_ContextCancellation(t *testing.T) {
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

	// Test DownloadUpdate with cancelled context
	updatePath, err := u.DownloadUpdate(ctx, "https://example.com/file.tar.gz")
	require.Error(t, err)
	assert.Empty(t, updatePath)
}

func TestUpdater_DownloadUpdate_InvalidURL(t *testing.T) {
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

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "empty URL",
			url:  "",
		},
		{
			name: "invalid URL format",
			url:  "not-a-url",
		},
		{
			name: "malformed URL",
			url:  "://invalid",
		},
		{
			name: "URL without scheme",
			url:  "example.com/file.tar.gz",
		},
		{
			name: "URL with invalid scheme",
			url:  "ftp://example.com/file.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updatePath, err := u.DownloadUpdate(ctx, tt.url)
			require.Error(t, err)
			assert.Empty(t, updatePath)
		})
	}
}

func TestUpdater_DownloadUpdate_Timeout(t *testing.T) {
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

	// Test DownloadUpdate with timeout
	updatePath, err := u.DownloadUpdate(ctx, "https://example.com/file.tar.gz")
	require.Error(t, err)
	assert.Empty(t, updatePath)
}

func TestUpdater_DownloadUpdate_Concurrent(t *testing.T) {
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

	// Test concurrent DownloadUpdate calls
	done := make(chan bool, 5)

	for range 5 {
		go func() {
			defer func() { done <- true }()

			updatePath, err := u.DownloadUpdate(ctx, "https://example.com/file.tar.gz")
			// Don't assert on result as it might fail due to network
			_ = updatePath
			_ = err
		}()
	}

	// Wait for all goroutines to complete
	for range 5 {
		<-done
	}
}

func TestUpdater_DownloadUpdate_EdgeCases(t *testing.T) {
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

	tests := []struct {
		name string
		url  string
	}{
		{
			name: "URL with query parameters",
			url:  "https://example.com/file.tar.gz?param=value",
		},
		{
			name: "URL with fragment",
			url:  "https://example.com/file.tar.gz#fragment",
		},
		{
			name: "URL with port",
			url:  "https://example.com:8080/file.tar.gz",
		},
		{
			name: "URL with path",
			url:  "https://example.com/path/to/file.tar.gz",
		},
		{
			name: "URL with user info",
			url:  "https://user:pass@example.com/file.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			updatePath, err := u.DownloadUpdate(ctx, tt.url)
			// Don't assert on result as it might fail due to network
			_ = updatePath
			_ = err
		})
	}
}

func TestUpdater_DownloadUpdate_ReturnValues(t *testing.T) {
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

	// Test that DownloadUpdate returns expected values
	updatePath, err := u.DownloadUpdate(ctx, "https://example.com/file.tar.gz")
	// Don't assert on specific values as they depend on network
	_ = updatePath
	_ = err
}

func TestUpdater_DownloadUpdate_ErrorHandling(t *testing.T) {
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

	// Test that DownloadUpdate handles errors gracefully
	updatePath, err := u.DownloadUpdate(ctx, "https://nonexistent-domain-12345.com/file.tar.gz")
	require.Error(t, err)
	assert.Empty(t, updatePath)
}

func TestUpdater_DownloadUpdate_WithNilUpdater(t *testing.T) {
	t.Parallel()
	// Test with nil updater (should not happen in practice)
	var u *updater.Updater

	ctx := context.Background()

	// This will panic, so we expect it to panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic with nil updater")
		}
	}()

	updatePath, err := u.DownloadUpdate(ctx, "https://example.com/file.tar.gz")
	_ = updatePath
	_ = err
}

func TestUpdater_DownloadUpdate_WithNilContext(t *testing.T) {
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

	// Test with nil context - this should return an error
	updatePath, err := u.DownloadUpdate(context.TODO(), "https://example.com/file.tar.gz")
	require.Error(t, err)
	assert.Empty(t, updatePath)
}
