//nolint:gochecknoglobals // version info set via ldflags
package version

// These variables are intended to be set via -ldflags at build time.
// Example:
//
//	-X github.com/bavix/outway/internal/version.Version=v1.2.3 \
//	-X github.com/bavix/outway/internal/version.BuildTime=2025-09-24T12:00:00Z
var (
	Version   = "dev"
	BuildTime = ""
)

func GetVersion() string { return Version }

func GetBuildTime() string { return BuildTime }
