package updater_test

import (
	"github.com/bavix/outway/internal/updater"
)

// getBasicConfigTests returns basic configuration test cases.
//
//nolint:funlen
func getBasicConfigTests() []struct {
	name    string
	config  updater.Config
	wantErr bool
} {
	return []struct {
		name    string
		config  updater.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: "v1.0.0",
				BinaryName:     "test",
			},
			wantErr: false,
		},
		{
			name: "valid config with custom values",
			config: updater.Config{
				Owner:          "custom-owner",
				Repo:           "custom-repo",
				CurrentVersion: "v2.1.0",
				BinaryName:     "custom-binary",
			},
			wantErr: false,
		},
		{
			name: "invalid config - empty owner",
			config: updater.Config{
				Owner:          "",
				Repo:           "test",
				CurrentVersion: "v1.0.0",
				BinaryName:     "test",
			},
			wantErr: true,
		},
		{
			name: "invalid config - empty repo",
			config: updater.Config{
				Owner:          "test",
				Repo:           "",
				CurrentVersion: "v1.0.0",
				BinaryName:     "test",
			},
			wantErr: true,
		},
		{
			name: "invalid config - empty current version",
			config: updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: "",
				BinaryName:     "test",
			},
			wantErr: true,
		},
		{
			name: "invalid config - empty binary name",
			config: updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: "v1.0.0",
				BinaryName:     "",
			},
			wantErr: true,
		},
		{
			name: "valid version without v prefix",
			config: updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: "1.0.0",
				BinaryName:     "test",
			},
			wantErr: false, // This should be valid as it gets normalized
		},
	}
}

// getEdgeCaseConfigTests returns edge case configuration test cases.
func getEdgeCaseConfigTests() []struct {
	name    string
	config  updater.Config
	wantErr bool
} {
	return getSpecialCharacterConfigTests()
}

// getSpecialCharacterConfigTests returns special character configuration test cases.
//
//nolint:funlen
func getSpecialCharacterConfigTests() []struct {
	name    string
	config  updater.Config
	wantErr bool
} {
	return []struct {
		name    string
		config  updater.Config
		wantErr bool
	}{
		{
			name: "owner with hyphens",
			config: updater.Config{
				Owner:          "test-owner",
				Repo:           "test",
				CurrentVersion: "v1.0.0",
				BinaryName:     "test",
			},
			wantErr: false,
		},
		{
			name: "repo with hyphens",
			config: updater.Config{
				Owner:          "test",
				Repo:           "test-repo",
				CurrentVersion: "v1.0.0",
				BinaryName:     "test",
			},
			wantErr: false,
		},
		{
			name: "binary name with hyphens",
			config: updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: "v1.0.0",
				BinaryName:     "test-binary",
			},
			wantErr: false,
		},
		{
			name: "owner with underscores",
			config: updater.Config{
				Owner:          "test_owner",
				Repo:           "test",
				CurrentVersion: "v1.0.0",
				BinaryName:     "test",
			},
			wantErr: false,
		},
		{
			name: "repo with underscores",
			config: updater.Config{
				Owner:          "test",
				Repo:           "test_repo",
				CurrentVersion: "v1.0.0",
				BinaryName:     "test",
			},
			wantErr: false,
		},
		{
			name: "binary name with underscores",
			config: updater.Config{
				Owner:          "test",
				Repo:           "test",
				CurrentVersion: "v1.0.0",
				BinaryName:     "test_binary",
			},
			wantErr: false,
		},
	}
}
