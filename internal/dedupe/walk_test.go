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

func TestDedupeContainedRootsHandlesLexicallyInterleavedSibling(t *testing.T) {
	base := string(filepath.Separator) + "base"
	a := filepath.Join(base, "a")
	aBang := a + "!"
	sub := filepath.Join(a, "sub")

	// "a!" sorts between "a" and "a/sub" lexically; a naive "compare only
	// against the previously kept root" check would let "sub" slip through
	// as a redundant, separately-walked root instead of being recognized as
	// already covered by "a".
	got := dedupeContainedRoots([]string{a, aBang, sub})
	require.ElementsMatch(t, []string{a, aBang}, got)
}

func TestWalkDirsContainmentSurvivesLexicallyInterleavedSibling(t *testing.T) {
	base := t.TempDir()
	a := filepath.Join(base, "a")
	aBang := filepath.Join(base, "a!")
	sub := filepath.Join(a, "sub")
	require.NoError(t, os.Mkdir(a, 0o755))
	require.NoError(t, os.Mkdir(aBang, 0o755))
	require.NoError(t, os.Mkdir(sub, 0o755))
	writeFile(t, filepath.Join(a, "a.txt"), "aaa")
	writeFile(t, filepath.Join(sub, "b.txt"), "bb")
	writeFile(t, filepath.Join(aBang, "c.txt"), "cc")

	files, err := WalkDirs([]string{a, aBang, sub}, zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, files, 3)
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

	// The symlink is an intermediate path component here (not the root's final
	// component), so this covers canonicalRoot resolving roots that differ
	// only by an intermediate symlink — distinct from the directory-symlink-
	// as-root case covered by TestWalkDirsResolvesDirectorySymlinkRoot below.
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

func TestWalkDirsResolvesDirectorySymlinkRoot(t *testing.T) {
	realDir := t.TempDir()
	writeFile(t, filepath.Join(realDir, "a.txt"), "aaa")

	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(realDir, link); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}

	// filepath.Walk lstats (never dereferences) the root it's given, so a
	// directory symlink passed directly as a root must be resolved by
	// WalkDirs itself before walking, or it would silently scan nothing.
	files, err := WalkDirs([]string{link}, zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, files, 1)

	files, err = WalkDirs([]string{realDir, link}, zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestWalkDirsDanglingSymlinkRootReturnsError(t *testing.T) {
	missingTarget := filepath.Join(t.TempDir(), "does-not-exist")
	link := filepath.Join(t.TempDir(), "dangling")
	if err := os.Symlink(missingTarget, link); err != nil {
		t.Skipf("symlinks not supported: %v", err)
	}

	// A dangling root symlink must surface as an error, not a silent
	// zero-file success: filepath.Walk would otherwise Lstat the symlink
	// itself, see it's non-regular, and skip it without complaint.
	files, err := WalkDirs([]string{link}, zerolog.Nop())
	require.Error(t, err)
	require.Empty(t, files)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
