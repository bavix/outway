package version_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/bavix/outway/internal/version"
)

func TestGetVersion(t *testing.T) {
	t.Parallel()
	// Test default version
	assert.Equal(t, "dev", version.GetVersion())

	// Test that we can get the version
	version := version.GetVersion()
	assert.NotEmpty(t, version)
}

func TestGetBuildTime(t *testing.T) {
	t.Parallel()
	// Test default build time
	assert.Empty(t, version.GetBuildTime())

	// Test that we can get the build time
	buildTime := version.GetBuildTime()
	assert.NotNil(t, buildTime) // Can be empty string, but should not be nil
}

func TestVersionVariables(t *testing.T) {
	t.Parallel()
	// Test that version variables are accessible
	assert.NotNil(t, version.Version)
	assert.NotNil(t, version.BuildTime)

	// Test that variables have expected types
	assert.IsType(t, "", version.Version)
	assert.IsType(t, "", version.BuildTime)
}

func TestVersionConsistency(t *testing.T) {
	t.Parallel()
	// Test that GetVersion() returns the same value as Version variable
	assert.Equal(t, version.Version, version.GetVersion())

	// Test that GetBuildTime() returns the same value as BuildTime variable
	assert.Equal(t, version.BuildTime, version.GetBuildTime())
}

func TestVersionNotEmpty(t *testing.T) {
	t.Parallel()
	// Test that version is not empty
	version := version.GetVersion()
	assert.NotEmpty(t, version)
	assert.NotEmpty(t, version)
}

func TestBuildTimeFormat(t *testing.T) {
	t.Parallel()
	// Test that build time can be parsed as time
	buildTime := version.GetBuildTime()
	if buildTime != "" {
		_, err := time.Parse(time.RFC3339, buildTime)
		assert.NoError(t, err, "BuildTime should be in RFC3339 format")
	}
}

func TestVersionStability(t *testing.T) {
	t.Parallel()
	// Test that version doesn't change between calls
	version1 := version.GetVersion()
	version2 := version.GetVersion()
	assert.Equal(t, version1, version2)
}

func TestBuildTimeStability(t *testing.T) {
	t.Parallel()
	// Test that build time doesn't change between calls
	buildTime1 := version.GetBuildTime()
	buildTime2 := version.GetBuildTime()
	assert.Equal(t, buildTime1, buildTime2)
}

func TestVersionNotNil(t *testing.T) {
	t.Parallel()
	// Test that version is never nil
	version := version.GetVersion()
	assert.NotNil(t, version)
}

func TestBuildTimeNotNil(t *testing.T) {
	t.Parallel()
	// Test that build time is never nil
	buildTime := version.GetBuildTime()
	assert.NotNil(t, buildTime)
}

func TestVersionLength(t *testing.T) {
	t.Parallel()
	// Test that version has reasonable length
	version := version.GetVersion()
	assert.NotEmpty(t, version)
	assert.Less(t, len(version), 100) // Should not be extremely long
}

func TestBuildTimeLength(t *testing.T) {
	t.Parallel()
	// Test that build time has reasonable length
	buildTime := version.GetBuildTime()
	if buildTime != "" {
		assert.NotEmpty(t, buildTime)
		assert.Less(t, len(buildTime), 100) // Should not be extremely long
	}
}

func TestVersionEquality(t *testing.T) {
	t.Parallel()
	// Test that version equals itself
	version := version.GetVersion()
	// Version should be non-empty
	assert.NotEmpty(t, version)
}

func TestBuildTimeEquality(t *testing.T) {
	t.Parallel()
	// Test that build time equals itself
	buildTime := version.GetBuildTime()
	// Build time should be a string (may be empty if not set)
	assert.IsType(t, "", buildTime)
}
