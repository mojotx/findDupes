// Package version exposes the application version, set by the linker at release build time.
package version

// findDupesVersion is the version reported by the application. Release builds
// set it with Go linker flags.
var findDupesVersion = "dev"

// Version returns the current version of the application.
func Version() string {
	return findDupesVersion
}
