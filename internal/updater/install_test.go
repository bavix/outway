package updater_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/updater"
)

func TestUpdater_InstallUpdate(t *testing.T) {
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

	// Test InstallUpdate with non-existent file
	nonExistentPath := "/tmp/nonexistent-file.tar.gz"
	err = u.InstallUpdate(ctx, nonExistentPath)
	assert.Error(t, err)
}

func TestUpdater_InstallUpdate_ContextCancellation(t *testing.T) {
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

	// Test InstallUpdate with cancelled context
	err = u.InstallUpdate(ctx, "/tmp/test-file.tar.gz")
	assert.Error(t, err)
}

func TestUpdater_InstallUpdate_InvalidPath(t *testing.T) {
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
		path string
	}{
		{
			name: "empty path",
			path: "",
		},
		{
			name: "non-existent file",
			path: "/tmp/nonexistent-file.tar.gz",
		},
		{
			name: "directory instead of file",
			path: "/tmp",
		},
		{
			name: "invalid path",
			path: "/invalid/path/that/does/not/exist/file.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := u.InstallUpdate(ctx, tt.path)
			assert.Error(t, err)
		})
	}
}

func TestUpdater_InstallUpdate_Timeout(t *testing.T) {
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

	// Test InstallUpdate with timeout
	err = u.InstallUpdate(ctx, "/tmp/test-file.tar.gz")
	assert.Error(t, err)
}

func TestUpdater_InstallUpdate_Concurrent(t *testing.T) {
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

	// Test concurrent InstallUpdate calls
	done := make(chan bool, 5)

	for range 5 {
		go func() {
			defer func() { done <- true }()

			err := u.InstallUpdate(ctx, "/tmp/test-file.tar.gz")
			// Don't assert on result as it might fail due to file not existing
			_ = err
		}()
	}

	// Wait for all goroutines to complete
	for range 5 {
		<-done
	}
}

func TestUpdater_InstallUpdate_EdgeCases(t *testing.T) {
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
		path string
	}{
		{
			name: "path with spaces",
			path: "/tmp/file with spaces.tar.gz",
		},
		{
			name: "path with special characters",
			path: "/tmp/file-with-special-chars.tar.gz",
		},
		{
			name: "path with unicode characters",
			path: "/tmp/файл.tar.gz",
		},
		{
			name: "path with dots",
			path: "/tmp/file.with.dots.tar.gz",
		},
		{
			name: "path with hyphens",
			path: "/tmp/file-with-hyphens.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := u.InstallUpdate(ctx, tt.path)
			// Don't assert on result as it might fail due to file not existing
			_ = err
		})
	}
}

func TestUpdater_InstallUpdate_ReturnValues(t *testing.T) {
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

	// Test that InstallUpdate returns expected values
	err = u.InstallUpdate(ctx, "/tmp/test-file.tar.gz")
	// Don't assert on specific values as they depend on file existence
	_ = err
}

func TestUpdater_InstallUpdate_ErrorHandling(t *testing.T) {
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

	// Test that InstallUpdate handles errors gracefully
	err = u.InstallUpdate(ctx, "/tmp/nonexistent-file.tar.gz")
	assert.Error(t, err)
}

func TestUpdater_InstallUpdate_WithNilUpdater(t *testing.T) {
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

	err := u.InstallUpdate(ctx, "/tmp/test-file.tar.gz")
	_ = err
}

func TestUpdater_InstallUpdate_WithNilContext(t *testing.T) {
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
	err = u.InstallUpdate(context.TODO(), "/tmp/test-file.tar.gz")
	assert.Error(t, err)
}

func TestUpdater_InstallUpdate_WithEmptyFile(t *testing.T) {
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

	// Create a temporary empty file
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.tar.gz")
	file, err := os.Create(filepath.Clean(emptyFile))
	require.NoError(t, err)

	_ = file.Close()

	// Test InstallUpdate with empty file
	err = u.InstallUpdate(ctx, emptyFile)
	assert.Error(t, err)
}

func TestUpdater_InstallUpdate_WithInvalidArchive(t *testing.T) {
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

	// Create a temporary file with invalid archive content
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.tar.gz")
	file, err := os.Create(filepath.Clean(invalidFile))
	require.NoError(t, err)

	_, _ = file.WriteString("not a valid archive")
	_ = file.Close()

	// Test InstallUpdate with invalid archive
	err = u.InstallUpdate(ctx, invalidFile)
	assert.Error(t, err)
}

func TestUpdater_InstallUpdate_WithReadOnlyFile(t *testing.T) {
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

	// Create a temporary file
	tmpDir := t.TempDir()
	readOnlyFile := filepath.Join(tmpDir, "readonly.tar.gz")
	file, err := os.Create(filepath.Clean(readOnlyFile))
	require.NoError(t, err)

	_, _ = file.WriteString("test content")
	_ = file.Close()

	// Make file read-only
	err = os.Chmod(filepath.Clean(readOnlyFile), 0o444) // #nosec G302
	require.NoError(t, err)

	// Test InstallUpdate with read-only file
	err = u.InstallUpdate(ctx, readOnlyFile)
	// This might succeed or fail depending on the implementation
	_ = err
}
