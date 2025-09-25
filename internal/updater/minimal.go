// Package updater provides a minimal GitHub release updater for Go applications.
//
// Example usage:
//
//	config := updater.Config{
//		Owner:          "myorg",
//		Repo:           "myapp",
//		CurrentVersion: "v1.0.0",
//		BinaryName:     "myapp",
//	}
//
//	u := updater.New(config)
//	updateInfo, err := u.CheckForUpdates(ctx, false)
//	if err != nil {
//		return err
//	}
//
//	if updateInfo.HasUpdate {
//		updatePath, err := u.DownloadUpdate(ctx, updateInfo.Release.Assets[0].BrowserDownloadURL)
//		if err != nil {
//			return err
//		}
//		return u.InstallUpdate(ctx, updatePath)
//	}
package updater

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

// Logger defines a simple logging interface.
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// Config contains configuration for the updater.
type Config struct {
	Owner          string       // GitHub repository owner
	Repo           string       // GitHub repository name
	CurrentVersion string       // Current version of the application
	BinaryName     string       // Name of the binary to look for
	HTTPClient     *http.Client // Optional custom HTTP client
	Logger         Logger       // Optional logger
}

// Updater provides methods for checking, downloading, and installing updates.
type Updater struct {
	config     Config
	httpClient *http.Client
	logger     Logger
}

// Release represents a GitHub release.
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
	Assets      []Asset   `json:"assets"`
}

// Asset represents a file attached to a GitHub release.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	ContentType        string `json:"content_type"`
}

// UpdateInfo contains information about available updates.
type UpdateInfo struct {
	CurrentVersion string   `json:"current_version"`
	LatestVersion  string   `json:"latest_version"`
	Release        *Release `json:"release,omitempty"`
	HasUpdate      bool     `json:"has_update"`
}

// Predefined errors.
var (
	ErrInvalidConfig      = errors.New("invalid updater configuration")
	ErrNoReleasesFound    = errors.New("no releases found")
	ErrUpdateVerification = errors.New("update verification failed")
	ErrInstallationFailed = errors.New("installation failed")
	ErrDownloadFailed     = errors.New("download failed")
	ErrAPIError           = errors.New("GitHub API error")
)

const (
	defaultDownloadTimeout = 5 * time.Minute
	defaultAPITimeout      = 30 * time.Second
)

const (
	fileModeExec    = 0o755 // executable permission for downloaded/extracted binaries
	exitCodeRestart = 42    // special exit code to signal restart
)

// New creates a new updater instance.
func New(config Config) (*Updater, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	// Set up HTTP client
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultDownloadTimeout}
	}

	// Set up logger
	logger := config.Logger
	if logger == nil {
		logger = &noOpLogger{}
	}

	return &Updater{
		config:     config,
		httpClient: httpClient,
		logger:     logger,
	}, nil
}

// validateConfig validates the updater configuration.
func validateConfig(config Config) error {
	if config.Owner == "" {
		return errors.New("owner cannot be empty") //nolint:err113
	}

	if config.Repo == "" {
		return errors.New("repo cannot be empty") //nolint:err113
	}

	if config.CurrentVersion == "" {
		return errors.New("current version cannot be empty") //nolint:err113
	}

	if config.BinaryName == "" {
		return errors.New("binary name cannot be empty") //nolint:err113
	}

	return nil
}

// CheckForUpdates checks if there are newer releases available.
func (u *Updater) CheckForUpdates(ctx context.Context, includePrerelease bool) (*UpdateInfo, error) {
	u.logger.Debugf("checking for updates (include_prerelease=%t, current_version=%s)",
		includePrerelease, u.config.CurrentVersion)

	releases, err := u.fetchReleases(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAPIError, err)
	}

	if len(releases) == 0 {
		return nil, ErrNoReleasesFound
	}

	// Find the latest suitable release
	var latestRelease *Release

	for _, release := range releases {
		if release.Draft {
			continue
		}

		if !includePrerelease && release.Prerelease {
			continue
		}

		latestRelease = &release

		break
	}

	if latestRelease == nil {
		return &UpdateInfo{
			CurrentVersion: u.config.CurrentVersion,
			LatestVersion:  u.config.CurrentVersion,
			HasUpdate:      false,
		}, nil
	}

	hasUpdate := u.IsNewerVersion(latestRelease.TagName, u.config.CurrentVersion)

	updateInfo := &UpdateInfo{
		CurrentVersion: u.config.CurrentVersion,
		LatestVersion:  latestRelease.TagName,
		Release:        latestRelease,
		HasUpdate:      hasUpdate,
	}

	u.logger.Infof("update check completed (current=%s, latest=%s, has_update=%t)",
		updateInfo.CurrentVersion, updateInfo.LatestVersion, updateInfo.HasUpdate)

	return updateInfo, nil
}

// DownloadUpdate downloads the specified asset to a temporary location.

func (u *Updater) DownloadUpdate(ctx context.Context, downloadURL string) (string, error) {
	u.logger.Infof("downloading update from %s", downloadURL)

	if downloadURL == "" {
		return "", fmt.Errorf("%w: download URL cannot be empty", ErrDownloadFailed)
	}

	// Download to temporary file
	tmpFile, err := u.downloadToTemp(ctx, downloadURL)
	if err != nil {
		return "", err
	}

	defer func() {
		if err != nil {
			_ = os.Remove(tmpFile.Name())
		}

		_ = tmpFile.Close()
	}()

	// Extract and process the downloaded file
	path, err := u.extractAndProcess(tmpFile.Name())
	if err != nil {
		return "", err
	}

	return path, nil
}

// InstallUpdate installs the downloaded update by replacing the current binary.
// After successful installation, the application will exit with code 42.
func (u *Updater) InstallUpdate(ctx context.Context, updatePath string) error {
	u.logger.Infof("installing update from %s", updatePath)

	if updatePath == "" {
		return fmt.Errorf("%w: update path cannot be empty", ErrInstallationFailed)
	}

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("%w: failed to get executable path: %w", ErrInstallationFailed, err)
	}

	// Verify the update file
	if err := u.verifyUpdate(ctx, updatePath); err != nil {
		return fmt.Errorf("%w: %w", ErrUpdateVerification, err)
	}

	// Create backup of current executable
	backupPath := execPath + ".backup"
	if err := u.copyFile(execPath, backupPath); err != nil {
		return fmt.Errorf("%w: failed to create backup: %w", ErrInstallationFailed, err)
	}

	// Replace current binary with new one
	if err := u.copyFile(updatePath, execPath); err != nil {
		// Restore backup on failure
		_ = u.copyFile(backupPath, execPath)
		_ = os.Remove(backupPath)

		return fmt.Errorf("%w: failed to replace binary: %w", ErrInstallationFailed, err)
	}

	// Set executable permissions on new binary
	if err := os.Chmod(execPath, fileModeExec); err != nil {
		// Restore backup on failure
		_ = u.copyFile(backupPath, execPath)
		_ = os.Remove(backupPath)

		return fmt.Errorf("%w: failed to set executable permissions: %w", ErrInstallationFailed, err)
	}

	// Clean up
	_ = os.Remove(backupPath)
	_ = os.Remove(updatePath)

	u.logger.Infof("update installed successfully to %s, exiting for restart", execPath)

	// Exit with special code to trigger restart by init system
	os.Exit(exitCodeRestart)

	return nil // This line will never be reached
}

// IsNewerVersion compares two version strings.
func (u *Updater) IsNewerVersion(current, latest string) bool {
	// semver.Compare expects leading 'v' and proper semver strings
	ensureV := func(v string) string {
		if v == "" {
			return v
		}

		if v[0] != 'v' {
			return "v" + v
		}

		return v
	}

	c := ensureV(current)
	l := ensureV(latest)

	return semver.Compare(c, l) > 0
}

// downloadToTemp downloads the file to a temporary location.
func (u *Updater) downloadToTemp(ctx context.Context, downloadURL string) (*os.File, error) {
	tmpFile, err := os.CreateTemp("", u.config.BinaryName+"-update-*")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create temp file: %w", ErrDownloadFailed, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())

		return nil, fmt.Errorf("%w: failed to create request: %w", ErrDownloadFailed, err)
	}

	req.Header.Set("User-Agent", u.config.BinaryName+"-updater/1.0")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())

		return nil, fmt.Errorf("%w: failed to download: %w", ErrDownloadFailed, err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())

		return nil, fmt.Errorf("%w: HTTP %d", ErrDownloadFailed, resp.StatusCode)
	}

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())

		return nil, fmt.Errorf("%w: failed to write file: %w", ErrDownloadFailed, err)
	}

	u.logger.Infof("update downloaded successfully to %s (size=%d bytes)",
		tmpFile.Name(), resp.ContentLength)

	return tmpFile, nil
}

// extractAndProcess extracts archives and sets proper permissions.
func (u *Updater) extractAndProcess(filePath string) (string, error) {
	path := filePath

	extractors := []extractor{gzipExtractor{}, tarExtractor{u: u}, zipExtractor{u: u}}
	for _, ex := range extractors {
		out, ok, exErr := ex.Extract(path)
		if exErr != nil {
			return "", fmt.Errorf("%w: failed to extract archive: %w", ErrDownloadFailed, exErr)
		}

		if ok {
			_ = os.Remove(path) // remove previous stage
			path = out
		}
	}

	if err := os.Chmod(path, fileModeExec); err != nil {
		return "", fmt.Errorf("%w: failed to set permissions: %w", ErrDownloadFailed, err)
	}

	return path, nil
}

// trimVPrefix removed in favor of golang.org/x/mod/semver usage

// verifyUpdate verifies that the update file is valid and executable.
func (u *Updater) verifyUpdate(ctx context.Context, updatePath string) error {
	// Check if file exists
	if _, err := os.Stat(updatePath); err != nil {
		return fmt.Errorf("update file does not exist: %w", err)
	}

	// Try to execute the file with --version flag to verify it's a valid binary
	cmd := exec.CommandContext(ctx, updatePath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("update file is not executable or invalid: %w", err)
	}

	u.logger.Debugf("update file verified successfully: %s", updatePath)

	return nil
}

// copyFile copies a file from src to dst.
func (u *Updater) copyFile(src, dst string) error {
	srcFile, err := os.Open(src) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}

	defer func() { _ = srcFile.Close() }()

	dstFile, err := os.Create(dst) //nolint:gosec
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}

	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// fetchReleases fetches releases from the GitHub API.
func (u *Updater) fetchReleases(ctx context.Context) ([]Release, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", u.config.Owner, u.config.Repo)

	u.logger.Debugf("fetching releases from GitHub API: %s", apiURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", u.config.BinaryName+"-updater/1.0")

	client := &http.Client{Timeout: defaultAPITimeout}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", ErrAPIError, resp.StatusCode)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	u.logger.Debugf("fetched %d releases from GitHub API", len(releases))

	return releases, nil
}

// hasTARGZSuffix returns true if URL path suggests a .tar.gz archive.
func hasTARGZSuffix(raw string) bool { //nolint:unused // retained for compatibility
	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.HasSuffix(strings.ToLower(raw), ".tar.gz")
	}

	return strings.HasSuffix(strings.ToLower(parsed.Path), ".tar.gz")
}

// hasZipSuffix returns true if URL path suggests a .zip archive.
func hasZipSuffix(raw string) bool { //nolint:unused // retained for compatibility
	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.HasSuffix(strings.ToLower(raw), ".zip")
	}

	return strings.HasSuffix(strings.ToLower(parsed.Path), ".zip")
}

// extractor defines a step in the extraction pipeline.
// It either transforms the input path and returns (out,true), or passes through with ("",false).
type extractor interface {
	Extract(in string) (string, bool, error)
}

// gzipExtractor extracts .gz into a new temp file; non-gzip inputs pass through.
type gzipExtractor struct{}

func (gzipExtractor) Extract(in string) (string, bool, error) {
	if !strings.HasSuffix(strings.ToLower(in), ".gz") {
		f, err := os.Open(in) //nolint:gosec
		if err != nil {
			return "", false, nil
		}

		defer func() { _ = f.Close() }()

		if _, err = gzip.NewReader(f); err != nil {
			return "", false, nil
		}
	}

	f, err := os.Open(in) //nolint:gosec
	if err != nil {
		return "", false, err
	}

	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", false, nil
	}

	defer func() { _ = gz.Close() }()

	out, err := os.CreateTemp("", "updater-gz-*")
	if err != nil {
		return "", false, err
	}

	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, gz); err != nil { //nolint:gosec
		_ = os.Remove(out.Name())

		return "", false, err
	}

	return out.Name(), true, nil
}

// tarExtractor extracts a target binary from a tar stream; non-tar inputs pass through.
type tarExtractor struct{ u *Updater }

func (t tarExtractor) Extract(in string) (string, bool, error) {
	reader, err := t.openTarReader(in)
	if err != nil {
		return "", false, err
	}

	return t.extractFromTarReader(reader)
}

// openTarReader opens a tar reader, handling both .tar and .tar.gz files.
func (t tarExtractor) openTarReader(in string) (*tar.Reader, error) {
	f, err := os.Open(in) //nolint:gosec
	if err != nil {
		return nil, err
	}

	var r io.Reader = f
	if strings.HasSuffix(strings.ToLower(in), ".tar.gz") {
		if gz, gzErr := gzip.NewReader(f); gzErr == nil {
			defer func() { _ = gz.Close() }()

			r = gz
		}
	}

	return tar.NewReader(r), nil
}

// extractFromTarReader extracts the first matching binary from the tar reader.
func (t tarExtractor) extractFromTarReader(tr *tar.Reader) (string, bool, error) {
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return "", false, err
		}

		if hdr.FileInfo().IsDir() {
			continue
		}

		if !t.u.isTargetBinaryName(filepath.Base(hdr.Name)) {
			continue
		}

		return t.extractBinary(tr)
	}

	return "", false, nil
}

// extractBinary extracts a single binary file from the tar reader.
func (t tarExtractor) extractBinary(tr *tar.Reader) (string, bool, error) {
	tmpBin, err := os.CreateTemp("", t.u.config.BinaryName+"-bin-*")
	if err != nil {
		return "", false, err
	}

	if _, err := io.Copy(tmpBin, tr); err != nil {
		_ = tmpBin.Close()
		_ = os.Remove(tmpBin.Name())

		return "", false, err
	}

	_ = tmpBin.Close()

	if err := os.Chmod(tmpBin.Name(), fileModeExec); err != nil {
		_ = os.Remove(tmpBin.Name())

		return "", false, err
	}

	return tmpBin.Name(), true, nil
}

// zipExtractor extracts a target binary from a zip archive; non-zip inputs pass through.
type zipExtractor struct{ u *Updater }

func (z zipExtractor) Extract(in string) (string, bool, error) {
	zr, err := zip.OpenReader(in)
	if err != nil {
		return "", false, nil
	}

	defer func() { _ = zr.Close() }()

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}

		if !z.u.isTargetBinaryName(filepath.Base(f.Name)) {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return "", false, err
		}

		tmpBin, err := os.CreateTemp("", z.u.config.BinaryName+"-bin-*")
		if err != nil {
			_ = rc.Close()

			return "", false, err
		}

		if _, err := io.Copy(tmpBin, rc); err != nil { //nolint:gosec
			_ = rc.Close()
			_ = tmpBin.Close()
			_ = os.Remove(tmpBin.Name())

			return "", false, err
		}

		_ = rc.Close()

		_ = tmpBin.Close()
		if err := os.Chmod(tmpBin.Name(), fileModeExec); err != nil {
			_ = os.Remove(tmpBin.Name())

			return "", false, err
		}

		return tmpBin.Name(), true, nil
	}

	return "", false, nil
}

// isTargetBinaryName decides whether the provided name matches the expected binary
// It matches exact name or files that contain the binary name (common in release archives).
func (u *Updater) isTargetBinaryName(name string) bool {
	n := strings.ToLower(name)

	bn := strings.ToLower(u.config.BinaryName)
	if n == bn {
		return true
	}
	// Accept names that start with the binary name (e.g., myapp, myapp.exe)
	if strings.HasPrefix(n, bn) {
		// exclude obvious checksum or text files
		if strings.HasSuffix(n, ".sha256") || strings.HasSuffix(n, ".md5") || strings.HasSuffix(n, ".txt") || strings.HasSuffix(n, ".json") {
			return false
		}
		// exclude archives themselves
		if strings.HasSuffix(n, ".tar.gz") || strings.HasSuffix(n, ".zip") {
			return false
		}

		return true
	}

	return false
}

// noOpLogger is a no-op implementation of Logger interface.
type noOpLogger struct{}

func (n *noOpLogger) Debugf(format string, args ...any) {}
func (n *noOpLogger) Infof(format string, args ...any)  {}
func (n *noOpLogger) Warnf(format string, args ...any)  {}
func (n *noOpLogger) Errorf(format string, args ...any) {}
