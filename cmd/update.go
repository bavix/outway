package cmd

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/bavix/outway/internal/updater"
	verpkg "github.com/bavix/outway/internal/version"
)

var errNoSuitableAsset = errors.New("no suitable asset found for platform")

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "self-update",
		Short: "Update Outway to the latest version",
		Long:  getUpdateCommandLongDescription(),
		RunE:  runUpdateCommand,
	}

	// Add flags for prerelease updates
	cmd.Flags().BoolP("prerelease", "p", false, "Include prerelease versions")

	return cmd
}

// getUpdateCommandLongDescription returns the long description for the update command.
func getUpdateCommandLongDescription() string {
	return `Update Outway to the latest version from GitHub releases.

This command will:
1. Check for available updates on GitHub
2. Download the appropriate binary for your platform
3. Replace the current binary
4. Exit with code 42 to trigger automatic restart

The update process is safe and creates backups before replacing the binary.`
}

// runUpdateCommand executes the update command.
func runUpdateCommand(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	log := zerolog.Ctx(ctx)

	// Read flags
	prerelease, _ := cmd.Flags().GetBool("prerelease")

	// Create updater instance
	u, err := updater.New(updater.Config{
		Owner:          "bavix",
		Repo:           "outway",
		CurrentVersion: verpkg.GetVersion(),
		BinaryName:     "outway",
	})
	if err != nil {
		log.Err(err).Msg("failed to create updater")

		return fmt.Errorf("failed to create updater: %w", err)
	}

	log.Info().Msg("checking for updates...")

	// Check for updates
	updateInfo, err := u.CheckForUpdates(ctx, prerelease)
	if err != nil {
		log.Err(err).Msg("failed to check for updates")

		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if !updateInfo.HasUpdate {
		log.Info().Msg("Outway is already up to date")
		log.Info().Str("current_version", updateInfo.CurrentVersion).Msg("current version")

		// Log available version information even when no update is needed
		if updateInfo.Release != nil {
			log.Info().Str("latest_version", updateInfo.Release.TagName).Msg("latest version available")
			log.Info().Str("release_name", updateInfo.Release.Name).Msg("latest release")
		}

		return nil
	}

	return performUpdate(ctx, u, updateInfo)
}

// performUpdate handles the update process.
func performUpdate(ctx context.Context, u *updater.Updater, updateInfo *updater.UpdateInfo) error {
	log := zerolog.Ctx(ctx)
	logUpdateInfo(ctx, updateInfo)

	// Find appropriate asset for current platform
	asset := FindAssetForPlatform(updateInfo.Release.Assets, runtime.GOOS+"/"+runtime.GOARCH)
	if asset == nil {
		log.Error().
			Str("platform", runtime.GOOS+"/"+runtime.GOARCH).
			Msg("no suitable asset found for platform")

		return fmt.Errorf("%w %s/%s", errNoSuitableAsset, runtime.GOOS, runtime.GOARCH)
	}

	return downloadAndInstallUpdate(ctx, u, asset)
}

// logUpdateInfo logs information about the available update.
func logUpdateInfo(ctx context.Context, updateInfo *updater.UpdateInfo) {
	log := zerolog.Ctx(ctx)
	log.Info().Msg("Update available")
	log.Info().Str("current_version", updateInfo.CurrentVersion).Msg("current version")
	log.Info().Str("latest_version", updateInfo.LatestVersion).Msg("latest version")

	if updateInfo.Release != nil {
		log.Info().Str("release", updateInfo.Release.Name).Msg("release")

		if updateInfo.Release.Body != "" {
			log.Info().Str("release_notes", strings.TrimSpace(updateInfo.Release.Body)).Msg("release notes")
		}
	}
}

// downloadAndInstallUpdate downloads and installs the update.
func downloadAndInstallUpdate(ctx context.Context, u *updater.Updater, asset *updater.Asset) error {
	log := zerolog.Ctx(ctx)
	log.Info().Str("asset_name", asset.Name).Msg("downloading")
	log.Info().Str("size", FormatFileSize(asset.Size)).Msg("file size")

	// Download update
	updatePath, err := u.DownloadUpdate(ctx, asset.BrowserDownloadURL)
	if err != nil {
		log.Err(err).Msg("failed to download update")

		return fmt.Errorf("failed to download update: %w", err)
	}

	log.Info().Msg("Update downloaded successfully")

	// Install update
	log.Info().Msg("Installing update")

	err = u.InstallUpdate(ctx, updatePath)
	if err != nil {
		log.Err(err).Msg("failed to install update")

		return fmt.Errorf("failed to install update: %w", err)
	}

	log.Info().Msg("Update installed successfully!")

	return nil
}

// FindAssetForPlatform finds the appropriate asset for the platform.
func FindAssetForPlatform(assets []updater.Asset, platform string) *updater.Asset {
	// Extract OS and arch from platform string (e.g., "linux/amd64")
	parts := strings.Split(platform, "/")

	const expectedParts = 2
	if len(parts) != expectedParts {
		return nil
	}

	os, arch := parts[0], parts[1]

	// Try different matching strategies
	if asset := findExactMatch(assets, os, arch); asset != nil {
		return asset
	}

	if asset := findOSMatch(assets, os); asset != nil {
		return asset
	}

	return findBinaryAsset(assets)
}

// findExactMatch looks for exact platform match.
func findExactMatch(assets []updater.Asset, os, arch string) *updater.Asset {
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, strings.ToLower(os)) && strings.Contains(name, strings.ToLower(arch)) {
			return &asset
		}
	}

	return nil
}

// findOSMatch looks for OS match only.
func findOSMatch(assets []updater.Asset, os string) *updater.Asset {
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, strings.ToLower(os)) {
			return &asset
		}
	}

	return nil
}

// findBinaryAsset looks for any binary asset.
func findBinaryAsset(assets []updater.Asset) *updater.Asset {
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.Contains(name, ".tar.gz") || strings.Contains(name, ".zip") || strings.Contains(name, ".bin") {
			return &asset
		}
	}

	return nil
}

// FormatFileSize formats file size in human-readable format.
func FormatFileSize(bytes int64) string {
	if bytes == 0 {
		return "0 Bytes"
	}

	const (
		kbInt = int64(1024)
		k     = 1024.0
	)

	units := []string{"Bytes", "KB", "MB", "GB"}

	// Bytes case: no decimals
	if bytes < kbInt {
		return fmt.Sprintf("%d %s", bytes, units[0])
	}

	size := float64(bytes)

	unitIndex := 0
	for size >= k && unitIndex < len(units)-1 {
		size /= k
		unitIndex++
	}

	return fmt.Sprintf("%.2f %s", size, units[unitIndex])
}
