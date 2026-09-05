package version

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	assert.Equal(t, "dev", Version())
}

func TestVersionFromBuildInfo(t *testing.T) {
	buildInfo := &debug.BuildInfo{Main: debug.Module{Version: "v0.1.0"}}

	assert.Equal(t, "v0.1.0", versionFromBuildInfo("dev", buildInfo))
}

func TestVersionFromBuildInfoPreservesLinkerValue(t *testing.T) {
	buildInfo := &debug.BuildInfo{Main: debug.Module{Version: "v0.2.0"}}

	assert.Equal(t, "v0.1.0", versionFromBuildInfo("v0.1.0", buildInfo))
}
