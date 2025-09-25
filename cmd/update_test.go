package cmd_test

import (
	"testing"

	"github.com/bavix/outway/cmd"
	"github.com/bavix/outway/internal/updater"
)

func TestFindAssetForPlatform(t *testing.T) { //nolint:funlen // test function
	t.Parallel()

	tests := []struct {
		name     string
		assets   []string
		platform string
		expected string
	}{
		{
			name:     "exact platform match",
			assets:   []string{"outway_1.0.0_linux_amd64", "outway_1.0.0_darwin_amd64"},
			platform: "linux/amd64",
			expected: "outway_1.0.0_linux_amd64",
		},
		{
			name:     "os match",
			assets:   []string{"outway_1.0.0_linux_amd64", "outway_1.0.0_linux_arm64"},
			platform: "linux/amd64",
			expected: "outway_1.0.0_linux_amd64",
		},
		{
			name:     "no match",
			assets:   []string{"outway_1.0.0_darwin_amd64"},
			platform: "linux/amd64",
			expected: "",
		},
		{
			name:     "binary asset fallback",
			assets:   []string{"outway_1.0.0.tar.gz", "outway_1.0.0.zip"},
			platform: "linux/amd64",
			expected: "outway_1.0.0.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Convert string slice to Asset slice
			assets := make([]updater.Asset, len(tt.assets))
			for i, name := range tt.assets {
				assets[i] = updater.Asset{
					Name:               name,
					BrowserDownloadURL: "https://example.com/" + name,
					Size:               1024,
				}
			}

			result := cmd.FindAssetForPlatform(assets, tt.platform)

			if tt.expected == "" {
				if result != nil {
					t.Errorf("expected nil, got %v", result)
				}

				return
			}

			if result == nil {
				t.Errorf("expected asset %s, got nil", tt.expected)

				return
			}

			if result.Name != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result.Name)
			}
		})
	}
}

func TestFormatFileSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 Bytes"},
		{1023, "1023 Bytes"},
		{1024, "1.00 KB"},
		{1048576, "1.00 MB"},
		{1073741824, "1.00 GB"},
		{1536, "1.50 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			t.Parallel()

			result := cmd.FormatFileSize(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
