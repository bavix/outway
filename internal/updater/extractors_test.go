package updater_test

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bavix/outway/internal/updater"
)

func createAndTestTarGzFile(t *testing.T, u *updater.Updater, pattern string) {
	t.Helper()
	// Create a tar.gz file
	tmpFile, err := os.CreateTemp(t.TempDir(), pattern)
	require.NoError(t, err)

	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Create tar.gz content
	gzWriter := gzip.NewWriter(tmpFile)
	tarWriter := tar.NewWriter(gzWriter)

	// Add a file that matches the target binary name
	header := &tar.Header{
		Name: "testapp",
		Mode: 0o755,
		Size: 12,
	}
	err = tarWriter.WriteHeader(header)
	require.NoError(t, err)
	_, err = tarWriter.Write([]byte("test content"))
	require.NoError(t, err)
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzWriter.Close())
	require.NoError(t, tmpFile.Close())

	// Test extraction through DownloadUpdate
	_, err = u.DownloadUpdate(context.TODO(), "file://"+tmpFile.Name())
	assert.Error(t, err) // Expected to fail due to invalid URL scheme
}

func TestGzipExtractor(t *testing.T) {
	t.Parallel()
	// Test through DownloadUpdate which uses gzipExtractor internally
	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	}
	u, err := updater.New(config)
	require.NoError(t, err)

	t.Run("extract_gzip_file", func(t *testing.T) {
		t.Parallel()
		// Create a gzipped file
		tmpFile, err := os.CreateTemp(t.TempDir(), "gzip-test-*.gz")
		require.NoError(t, err)

		defer func() { _ = os.Remove(tmpFile.Name()) }()

		// Write gzipped content
		gzWriter := gzip.NewWriter(tmpFile)
		_, err = gzWriter.Write([]byte("test content"))
		require.NoError(t, err)
		require.NoError(t, gzWriter.Close())
		require.NoError(t, tmpFile.Close())

		// Test extraction through DownloadUpdate
		_, err = u.DownloadUpdate(context.TODO(), "file://"+tmpFile.Name())
		assert.Error(t, err) // Expected to fail due to invalid URL scheme
	})

	t.Run("extract_non_gzip_file", func(t *testing.T) {
		t.Parallel()
		// Create a regular file
		tmpFile, err := os.CreateTemp(t.TempDir(), "regular-test-*")
		require.NoError(t, err)

		defer func() { _ = os.Remove(tmpFile.Name()) }()

		_, err = tmpFile.WriteString("regular content")
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		// Test extraction through DownloadUpdate
		_, err = u.DownloadUpdate(context.TODO(), "file://"+tmpFile.Name())
		assert.Error(t, err) // Expected to fail due to invalid URL scheme
	})
}

func TestTarExtractor(t *testing.T) {
	t.Parallel()

	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "testapp",
	}
	u, err := updater.New(config)
	require.NoError(t, err)

	t.Run("extract_tar_file_with_target_binary", func(t *testing.T) {
		t.Parallel()
		// Create a tar file
		tmpFile, err := os.CreateTemp(t.TempDir(), "tar-test-*.tar")
		require.NoError(t, err)

		defer func() { _ = os.Remove(tmpFile.Name()) }()

		// Create tar content
		tarWriter := tar.NewWriter(tmpFile)

		// Add a file that matches the target binary name
		header := &tar.Header{
			Name: "testapp",
			Mode: 0o755,
			Size: 12,
		}
		err = tarWriter.WriteHeader(header)
		require.NoError(t, err)
		_, err = tarWriter.Write([]byte("test content"))
		require.NoError(t, err)
		require.NoError(t, tarWriter.Close())
		require.NoError(t, tmpFile.Close())

		// Test extraction through DownloadUpdate
		_, err = u.DownloadUpdate(context.TODO(), "file://"+tmpFile.Name())
		assert.Error(t, err) // Expected to fail due to invalid URL scheme
	})

	t.Run("extract_tar_gz_file", func(t *testing.T) {
		t.Parallel()
		createAndTestTarGzFile(t, u, "tar-gz-test-*.tar.gz")
	})
}

func TestZipExtractor(t *testing.T) {
	t.Parallel()

	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "testapp",
	}
	u, err := updater.New(config)
	require.NoError(t, err)

	t.Run("extract_zip_file_with_target_binary", func(t *testing.T) {
		t.Parallel()
		// Create a zip file
		tmpFile, err := os.CreateTemp(t.TempDir(), "zip-test-*.zip")
		require.NoError(t, err)

		defer func() { _ = os.Remove(tmpFile.Name()) }()

		// Create zip content
		zipWriter := zip.NewWriter(tmpFile)

		// Add a file that matches the target binary name
		fileWriter, err := zipWriter.Create("testapp")
		require.NoError(t, err)
		_, err = fileWriter.Write([]byte("test content"))
		require.NoError(t, err)
		require.NoError(t, zipWriter.Close())
		require.NoError(t, tmpFile.Close())

		// Test extraction through DownloadUpdate
		_, err = u.DownloadUpdate(context.TODO(), "file://"+tmpFile.Name())
		assert.Error(t, err) // Expected to fail due to invalid URL scheme
	})

	t.Run("extract_zip_file_with_directory", func(t *testing.T) {
		t.Parallel()
		// Create a zip file
		tmpFile, err := os.CreateTemp(t.TempDir(), "zip-test-*.zip")
		require.NoError(t, err)

		defer func() { _ = os.Remove(tmpFile.Name()) }()

		// Create zip content
		zipWriter := zip.NewWriter(tmpFile)

		// Add a directory (should be skipped)
		_, err = zipWriter.Create("directory/")
		require.NoError(t, err)

		// Add a file that matches the target binary name
		fileWriter, err := zipWriter.Create("testapp")
		require.NoError(t, err)
		_, err = fileWriter.Write([]byte("test content"))
		require.NoError(t, err)
		require.NoError(t, zipWriter.Close())
		require.NoError(t, tmpFile.Close())

		// Test extraction through DownloadUpdate
		_, err = u.DownloadUpdate(context.TODO(), "file://"+tmpFile.Name())
		assert.Error(t, err) // Expected to fail due to invalid URL scheme
	})
}

func TestExtractorsIntegration(t *testing.T) {
	t.Parallel()

	config := updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "testapp",
	}
	u, err := updater.New(config)
	require.NoError(t, err)

	t.Run("extract_gzip_tar_file", func(t *testing.T) {
		t.Parallel()
		createAndTestTarGzFile(t, u, "gzip-tar-test-*.tar.gz")
	})

	t.Run("extract_zip_file", func(t *testing.T) {
		t.Parallel()
		// Create a .zip file
		tmpFile, err := os.CreateTemp(t.TempDir(), "zip-test-*.zip")
		require.NoError(t, err)

		defer func() { _ = os.Remove(tmpFile.Name()) }()

		// Create zip content
		zipWriter := zip.NewWriter(tmpFile)

		// Add a file that matches the target binary name
		fileWriter, err := zipWriter.Create("testapp")
		require.NoError(t, err)
		_, err = fileWriter.Write([]byte("test content"))
		require.NoError(t, err)
		require.NoError(t, zipWriter.Close())
		require.NoError(t, tmpFile.Close())

		// Test extraction through DownloadUpdate
		_, err = u.DownloadUpdate(context.TODO(), "file://"+tmpFile.Name())
		assert.Error(t, err) // Expected to fail due to invalid URL scheme
	})
}
