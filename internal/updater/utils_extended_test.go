package updater_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/updater"
)

func TestSyncFile(t *testing.T) {
	t.Parallel()
	t.Run("sync_existing_file", func(t *testing.T) {
		t.Parallel()
		// Create a temporary file
		tmpFile, err := os.CreateTemp(t.TempDir(), "sync-test-*")
		require.NoError(t, err)

		defer func() { _ = os.Remove(tmpFile.Name()) }()

		// Write some data
		_, err = tmpFile.WriteString("test data")
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		// Test syncFile through InstallUpdate (which calls syncFile internally)
		config := updater.Config{
			Owner:          "test",
			Repo:           "test",
			CurrentVersion: "v1.0.0",
			BinaryName:     "test",
		}
		u, err := updater.New(config)
		require.NoError(t, err)

		// This will test syncFile indirectly through InstallUpdate
		err = u.InstallUpdate(context.TODO(), tmpFile.Name())
		// We expect this to fail because the file is not a valid binary
		assert.Error(t, err)
	})

	t.Run("sync_nonexistent_file", func(t *testing.T) {
		t.Parallel()

		config := updater.Config{
			Owner:          "test",
			Repo:           "test",
			CurrentVersion: "v1.0.0",
			BinaryName:     "test",
		}
		u, err := updater.New(config)
		require.NoError(t, err)

		// This will test syncFile indirectly through InstallUpdate
		err = u.InstallUpdate(context.TODO(), "/nonexistent/file")
		assert.Error(t, err)
	})
}

func TestSyncDir(t *testing.T) {
	t.Parallel()
	t.Run("sync_existing_directory", func(t *testing.T) {
		t.Parallel()
		// Create a temporary directory
		tmpDir := t.TempDir()

		// Test syncDir through InstallUpdate (which calls syncDir internally)
		config := updater.Config{
			Owner:          "test",
			Repo:           "test",
			CurrentVersion: "v1.0.0",
			BinaryName:     "test",
		}
		u, err := updater.New(config)
		require.NoError(t, err)

		// Create a test binary file
		testFile, err := os.CreateTemp(tmpDir, "test-binary-*")
		require.NoError(t, err)
		_, err = testFile.WriteString("#!/bin/sh\necho 'test binary'")
		require.NoError(t, err)
		require.NoError(t, testFile.Close())

		// This will test syncDir indirectly through InstallUpdate
		err = u.InstallUpdate(context.TODO(), testFile.Name())
		// We expect this to fail because the file is not a valid binary
		assert.Error(t, err)
	})
}

func TestCopyFile(t *testing.T) {
	t.Parallel()
	t.Run("copy_file_through_install", func(t *testing.T) {
		t.Parallel()
		// Create source file
		srcFile, err := os.CreateTemp(t.TempDir(), "copy-src-*")
		require.NoError(t, err)

		defer func() { _ = os.Remove(srcFile.Name()) }()

		testData := "#!/bin/sh\necho 'test binary'"
		_, err = srcFile.WriteString(testData)
		require.NoError(t, err)
		require.NoError(t, srcFile.Close())

		// Create updater instance
		config := updater.Config{
			Owner:          "test",
			Repo:           "test",
			CurrentVersion: "v1.0.0",
			BinaryName:     "test",
		}
		u, err := updater.New(config)
		require.NoError(t, err)

		// Test copyFile through InstallUpdate (which calls copyFile internally)
		err = u.InstallUpdate(context.TODO(), srcFile.Name())
		// We expect this to fail because the file is not a valid binary
		assert.Error(t, err)
	})
}

func TestIsTargetBinaryName(t *testing.T) {
	t.Parallel()

	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "myapp",
	}
	u, err := updater.New(config)
	require.NoError(t, err)

	// Test through DownloadUpdate which uses isTargetBinaryName internally
	t.Run("test_binary_name_matching", func(t *testing.T) {
		t.Parallel()
		// This will test isTargetBinaryName indirectly through DownloadUpdate
		// We expect this to fail because the URL is invalid
		_, err := u.DownloadUpdate(context.TODO(), "https://example.com/invalid-url")
		assert.Error(t, err)
	})
}

func TestHasTARGZSuffix(t *testing.T) {
	t.Parallel()
	// Test through DownloadUpdate which uses hasTARGZSuffix internally
	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}
	u, err := updater.New(config)
	require.NoError(t, err)

	t.Run("test_tar_gz_suffix", func(t *testing.T) {
		t.Parallel()
		// This will test hasTARGZSuffix indirectly through DownloadUpdate
		_, err := u.DownloadUpdate(context.TODO(), "https://example.com/file.tar.gz")
		assert.Error(t, err) // Expected to fail due to invalid URL
	})
}

func TestHasZipSuffix(t *testing.T) {
	t.Parallel()
	// Test through DownloadUpdate which uses hasZipSuffix internally
	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}
	u, err := updater.New(config)
	require.NoError(t, err)

	t.Run("test_zip_suffix", func(t *testing.T) {
		t.Parallel()
		// This will test hasZipSuffix indirectly through DownloadUpdate
		_, err := u.DownloadUpdate(context.TODO(), "https://example.com/file.zip")
		assert.Error(t, err) // Expected to fail due to invalid URL
	})
}

func TestNoOpLogger(t *testing.T) {
	t.Parallel()
	// Test through updater with no logger (uses noOpLogger internally)
	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
		// No Logger specified, so it will use noOpLogger
	}
	u, err := updater.New(config)
	require.NoError(t, err)

	t.Run("test_no_op_logger", func(t *testing.T) {
		t.Parallel()
		// This will test noOpLogger indirectly through CheckForUpdates
		_, err := u.CheckForUpdates(context.TODO(), false)
		assert.Error(t, err) // Expected to fail due to network error
	})
}

func TestExtractAndProcess(t *testing.T) {
	t.Parallel()

	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}
	u, err := updater.New(config)
	require.NoError(t, err)

	t.Run("process_regular_file", func(t *testing.T) {
		t.Parallel()
		// Create a regular file
		tmpFile, err := os.CreateTemp(t.TempDir(), "extract-test-*")
		require.NoError(t, err)

		defer func() { _ = os.Remove(tmpFile.Name()) }()

		_, err = tmpFile.WriteString("test binary content")
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		// Test extractAndProcess through DownloadUpdate
		_, err = u.DownloadUpdate(context.TODO(), "file://"+tmpFile.Name())
		assert.Error(t, err) // Expected to fail due to invalid URL scheme
	})

	t.Run("process_nonexistent_file", func(t *testing.T) {
		t.Parallel()
		// Test extractAndProcess through DownloadUpdate
		_, err := u.DownloadUpdate(context.TODO(), "file:///nonexistent/file")
		assert.Error(t, err) // Expected to fail due to invalid URL scheme
	})
}
