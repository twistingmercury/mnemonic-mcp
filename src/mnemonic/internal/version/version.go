// Package version provides build metadata information for the mnemonic service.
// Version variables are populated at build time via ldflags:
//
//	go build -ldflags "-X github.com/twistingmercury/mnemonic/internal/version.version=1.0.0 \
//	                   -X github.com/twistingmercury/mnemonic/internal/version.buildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
//	                   -X github.com/twistingmercury/mnemonic/internal/version.commit=$(git rev-parse --short HEAD)"
package version

var (
	version   = "n/a"
	buildDate = "n/a"
	commit    = "n/a"
)

// Version returns the semantic version of the build.
func Version() string {
	return version
}

// BuildDate returns the UTC timestamp when the build was created.
func BuildDate() string {
	return buildDate
}

// Commit returns the short git commit hash of the build.
func Commit() string {
	return commit
}

// Info returns all version information as a struct for structured responses.
type Info struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	Commit    string `json:"commit"`
}

// GetInfo returns all version information in a structured format.
func GetInfo() Info {
	return Info{
		Version:   version,
		BuildDate: buildDate,
		Commit:    commit,
	}
}
