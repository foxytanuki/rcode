// Package version provides build-time version information.
// Variables are set via -ldflags at build time.
package version

var (
	// Version is the semantic version (e.g., "v0.3.1" or "v0.3.1-3-g1234567").
	// Set via: -X github.com/foxytanuki/rcode/internal/version.Version=$(git describe --tags --always --dirty)
	Version = "v0.3.2"

	// BuildTime is the UTC timestamp when the binary was built.
	// Set via: -X github.com/foxytanuki/rcode/internal/version.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
	BuildTime = "unknown"

	// GitHash is the short git commit hash.
	// Set via: -X github.com/foxytanuki/rcode/internal/version.GitHash=$(git rev-parse --short HEAD)
	GitHash = "unknown"
)
