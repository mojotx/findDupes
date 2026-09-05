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

func TestWalkDirsDeduplicatesOverlappingRoots(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	require.NoError(t, os.Mkdir(sub, 0o755))
	writeFile(t, filepath.Join(dir, "a.txt"), "aaa")
	writeFile(t, filepath.Join(sub, "b.txt"), "bb")

	files, err := WalkDirs([]string{dir, dir}, zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, files, 2)

	files, err = WalkDirs([]string{dir, sub}, zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, files, 2)
}

func TestWalkDirsDeduplicatesRelativeAndAbsoluteRoots(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "aaa")

	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	files, err := WalkDirs([]string{".", dir}, zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestWalkDirsDeduplicatesSymlinkedRoots(t *testing.T) {
	base := t.TempDir()
	realDir := filepath.Join(base, "real")
	require.NoError(t, os.Mkdir(realDir, 0o755))
	writeFile(t, filepath.Join(realDir, "a.txt"), "aaa")

	// The symlink must be an intermediate path component, not the root's final
	// component: filepath.Walk lstats (never dereferences) the root itself, so
	// a root that IS a symlink would just be skipped as non-regular and the
	// test would pass vacuously regardless of EvalSymlinks. The OS resolves
	// symlinked intermediate components transparently, so this actually
	// exercises the dedup logic.
	outer := t.TempDir()
	linkedParent := filepath.Join(outer, "linked")
	if err := os.Symlink(base, linkedParent); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}
	root2 := filepath.Join(linkedParent, "real")

	files, err := WalkDirs([]string{realDir, root2}, zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
