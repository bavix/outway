package updater_test

import (
	"context"
	"testing"

	"github.com/bavix/outway/internal/updater"
)

func TestUpdater_CheckForUpdates(t *testing.T) {
	t.Parallel()

	u, err := updater.New(updater.Config{
		Owner:          "bavix",
		Repo:           "outway",
		CurrentVersion: "v1.0.0",
		BinaryName:     "outway",
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	updateInfo, err := u.CheckForUpdates(ctx, false)
	if err != nil {
		t.Logf("Error checking for updates (expected if no internet): %v", err)

		return
	}

	if updateInfo == nil {
		t.Fatal("UpdateInfo should not be nil")
	}

	if updateInfo.CurrentVersion != "v1.0.0" {
		t.Errorf("Expected current version v1.0.0, got %s", updateInfo.CurrentVersion)
	}

	t.Logf("Update check result: HasUpdate=%v, LatestVersion=%s",
		updateInfo.HasUpdate, updateInfo.LatestVersion)
}

func TestUpdater_isNewerVersion(t *testing.T) {
	t.Parallel()

	u, err := updater.New(updater.Config{
		Owner:          "test",
		Repo:           "test",
		CurrentVersion: "v1.0.0",
		BinaryName:     "test",
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		version1 string
		version2 string
		expected bool
	}{
		{"v1.0.1", "v1.0.0", true},
		{"v1.0.0", "v1.0.1", false},
		{"v2.0.0", "v1.9.9", true},
		{"v1.0.0", "v1.0.0", false},
		{"1.0.1", "1.0.0", true},
		{"2.0.0", "1.9.9", true},
	}

	for _, test := range tests {
		t.Run(test.version1+" vs "+test.version2, func(t *testing.T) {
			t.Parallel()

			result := u.IsNewerVersion(test.version1, test.version2)
			if result != test.expected {
				t.Errorf("isNewerVersion(%s, %s) = %v, expected %v",
					test.version1, test.version2, result, test.expected)
			}
		})
	}
}
