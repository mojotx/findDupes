package dedupe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestWalkDirsCollectsRegularFiles(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))
	writeFile(t, filepath.Join(dir, "a.txt"), "aaa")
	writeFile(t, filepath.Join(sub, "b.txt"), "bb")

	files, err := WalkDirs([]string{dir}, zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, files, 2)
}

func TestWalkDirsMissingRootIsReturnedAsError(t *testing.T) {
	files, err := WalkDirs([]string{filepath.Join(t.TempDir(), "does-not-exist")}, zerolog.Nop())
	require.Error(t, err)
	require.Empty(t, files)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
