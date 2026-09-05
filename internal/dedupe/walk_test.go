package dedupe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// walkDirsCase exercises WalkDirs end to end: setup builds a directory tree
// and returns the roots to pass in, and the assertions check the resulting
// error/file count. Cases that need to skip (e.g. no symlink support) call
// t.Skip from within setup.
type walkDirsCase struct {
	name      string
	setup     func(t *testing.T) []string
	wantErr   bool
	wantFiles int
}

func TestWalkDirs(t *testing.T) {
	cases := []walkDirsCase{
		{
			name: "collects regular files recursively",
			setup: func(t *testing.T) []string {
				dir := t.TempDir()
				sub := filepath.Join(dir, "sub")
				require.NoError(t, os.Mkdir(sub, 0o755))
				writeFile(t, filepath.Join(dir, "a.txt"), "aaa")
				writeFile(t, filepath.Join(sub, "b.txt"), "bb")
				return []string{dir}
			},
			wantFiles: 2,
		},
		{
			name: "missing root is returned as an error",
			setup: func(t *testing.T) []string {
				return []string{filepath.Join(t.TempDir(), "does-not-exist")}
			},
			wantErr: true,
		},
		{
			name: "exact duplicate roots are deduplicated",
			setup: func(t *testing.T) []string {
				dir := t.TempDir()
				sub := filepath.Join(dir, "sub")
				require.NoError(t, os.Mkdir(sub, 0o755))
				writeFile(t, filepath.Join(dir, "a.txt"), "aaa")
				writeFile(t, filepath.Join(sub, "b.txt"), "bb")
				return []string{dir, dir}
			},
			wantFiles: 2,
		},
		{
			name: "parent and child roots are deduplicated",
			setup: func(t *testing.T) []string {
				dir := t.TempDir()
				sub := filepath.Join(dir, "sub")
				require.NoError(t, os.Mkdir(sub, 0o755))
				writeFile(t, filepath.Join(dir, "a.txt"), "aaa")
				writeFile(t, filepath.Join(sub, "b.txt"), "bb")
				return []string{dir, sub}
			},
			wantFiles: 2,
		},
		{
			name: "containment check survives a lexically interleaved sibling",
			setup: func(t *testing.T) []string {
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
				// "a!" sorts between "a" and "a/sub" lexically; a naive
				// "compare only against the previously kept root" check
				// would let "sub" slip through as a redundant root.
				return []string{a, aBang, sub}
			},
			wantFiles: 3,
		},
		{
			name: "relative and absolute spellings of the same root are deduplicated",
			setup: func(t *testing.T) []string {
				dir := t.TempDir()
				writeFile(t, filepath.Join(dir, "a.txt"), "aaa")
				cwd, err := os.Getwd()
				require.NoError(t, err)
				t.Chdir(dir)
				t.Cleanup(func() { _ = os.Chdir(cwd) })
				return []string{".", dir}
			},
			wantFiles: 1,
		},
		{
			name: "root reached via an intermediate symlink is deduplicated against its real path",
			setup: func(t *testing.T) []string {
				base := t.TempDir()
				realDir := filepath.Join(base, "real")
				require.NoError(t, os.Mkdir(realDir, 0o755))
				writeFile(t, filepath.Join(realDir, "a.txt"), "aaa")
				outer := t.TempDir()
				linkedParent := filepath.Join(outer, "linked")
				if err := os.Symlink(base, linkedParent); err != nil {
					t.Skipf("symlinks not supported: %v", err)
				}
				return []string{realDir, filepath.Join(linkedParent, "real")}
			},
			wantFiles: 1,
		},
		{
			name: "directory symlink root is resolved and walked",
			setup: func(t *testing.T) []string {
				realDir := t.TempDir()
				writeFile(t, filepath.Join(realDir, "a.txt"), "aaa")
				link := filepath.Join(t.TempDir(), "link")
				if err := os.Symlink(realDir, link); err != nil {
					t.Skipf("symlinks not supported: %v", err)
				}
				return []string{link}
			},
			wantFiles: 1,
		},
		{
			name: "directory symlink root is deduplicated against its real path",
			setup: func(t *testing.T) []string {
				realDir := t.TempDir()
				writeFile(t, filepath.Join(realDir, "a.txt"), "aaa")
				link := filepath.Join(t.TempDir(), "link")
				if err := os.Symlink(realDir, link); err != nil {
					t.Skipf("symlinks not supported: %v", err)
				}
				return []string{realDir, link}
			},
			wantFiles: 1,
		},
		{
			// filepath.Walk lstats (never dereferences) the root it's given,
			// so a dangling root symlink must surface as an error rather
			// than a silent zero-file success.
			name: "dangling symlink root returns an error",
			setup: func(t *testing.T) []string {
				missingTarget := filepath.Join(t.TempDir(), "does-not-exist")
				link := filepath.Join(t.TempDir(), "dangling")
				if err := os.Symlink(missingTarget, link); err != nil {
					t.Skipf("symlinks not supported: %v", err)
				}
				return []string{link}
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			roots := tc.setup(t)
			files, err := WalkDirs(roots, zerolog.Nop())
			if tc.wantErr {
				require.Error(t, err)
				require.Empty(t, files)
				return
			}
			require.NoError(t, err)
			require.Len(t, files, tc.wantFiles)
		})
	}
}

func TestDedupeContainedRootsKeepsOneOfCaseInsensitiveDuplicateRoots(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "Data"), 0o755))

	upper := filepath.Join(dir, "Data")
	lower := filepath.Join(dir, "data")

	// Detect case-insensitivity empirically (macOS/APFS and Windows by
	// default; typically not Linux) rather than gating on GOOS, so this runs
	// for real wherever the filesystem actually behaves that way.
	upperInfo, err := os.Stat(upper)
	require.NoError(t, err)
	lowerInfo, err := os.Stat(lower)
	if err != nil || !os.SameFile(upperInfo, lowerInfo) {
		t.Skip("filesystem is case-sensitive")
	}

	got := dedupeContainedRoots([]string{upper, lower})
	require.Len(t, got, 1)
}

func TestDedupeContainedRootsHandlesLexicallyInterleavedSibling(t *testing.T) {
	base := t.TempDir()
	a := filepath.Join(base, "a")
	aBang := filepath.Join(base, "a!")
	sub := filepath.Join(a, "sub")
	require.NoError(t, os.Mkdir(a, 0o755))
	require.NoError(t, os.Mkdir(aBang, 0o755))
	require.NoError(t, os.Mkdir(sub, 0o755))

	// "a!" sorts between "a" and "a/sub" lexically; a naive "compare only
	// against the previously kept root" check would let "sub" slip through
	// as a redundant, separately-walked root instead of being recognized as
	// already covered by "a".
	got := dedupeContainedRoots([]string{a, aBang, sub})
	require.ElementsMatch(t, []string{a, aBang}, got)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
