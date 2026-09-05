// Package version exposes the application version, set by the linker at release build time.
package version

import "runtime/debug"

// findDupesVersion is the version reported by the application. Release builds
// set it with Go linker flags.
var findDupesVersion = "dev"

func init() {
	if findDupesVersion != "dev" {
		return
	}

	if buildInfo, ok := debug.ReadBuildInfo(); ok && buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		findDupesVersion = buildInfo.Main.Version
	}
}

// Version returns the current version of the application.
func Version() string {
	return findDupesVersion
}
