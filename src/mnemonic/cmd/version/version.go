// Package version provides CLI-specific version display functionality.
// It delegates to internal/version for the actual version values.
package version

import (
	"fmt"

	internalversion "github.com/twistingmercury/mnemonic/internal/version"
)

// Version returns the semantic version of the build.
func Version() string {
	return internalversion.Version()
}

// BuildDate returns the UTC timestamp when the build was created.
func BuildDate() string {
	return internalversion.BuildDate()
}

// Commit returns the short git commit hash of the build.
func Commit() string {
	return internalversion.Commit()
}

// Print returns a formatted ASCII art banner with version information
// suitable for CLI display.
func Print() string {
	const mnemonic = `
  __  __                                  _
 |  \/  |                                (_)
 | \  / |_ __   ___ _ __ ___   ___  _ __  _  ___
 | |\/| | '_ \ / _ \ '_ ' _ \ / _ \| '_ \| |/ __|
 | |  | | | | |  __/ | | | | | (_) | | | | | (__
 |_|  |_|_| |_|\___|_| |_| |_|\___/|_| |_|_|\___|`

	return fmt.Sprintf("%s\n                                   version %s\n", mnemonic, internalversion.Version())
}
