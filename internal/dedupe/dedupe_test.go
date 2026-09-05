package dedupe

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestHashAllClampsNonPositiveWorkers(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "aaa")
	candidates := []FileEntry{{Path: filepath.Join(dir, "a.txt"), Size: 3}}

	var results []Result
	for r := range hashAll(candidates, 0, zerolog.Nop()) {
		results = append(results, r)
	}
	require.Len(t, results, 1)
}

func TestHashAllSkipsUnhashableFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.txt")
	candidates := []FileEntry{{Path: missing, Size: 3}}

	var results []Result
	for r := range hashAll(candidates, 2, zerolog.Nop()) {
		results = append(results, r)
	}
	require.Empty(t, results)
}

func TestFilterBySize(t *testing.T) {
	files := []FileEntry{
		{Path: "a", Size: 10},
		{Path: "b", Size: 10},
		{Path: "c", Size: 20},
	}
	candidates, skipped := FilterBySize(files)
	require.Equal(t, 1, skipped)
	require.Len(t, candidates, 2)
}

func TestFind(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "dup1.txt"), "same content")
	writeFile(t, filepath.Join(dir, "dup2.txt"), "same content")
	writeFile(t, filepath.Join(dir, "unique.txt"), "one of a kind")

	dupes, stats, err := Find([]string{dir}, 2, zerolog.Nop())
	require.NoError(t, err)
	require.Equal(t, 3, stats.TotalFiles)
	require.Equal(t, 1, stats.Skipped)
	require.Len(t, dupes, 1)
	require.Len(t, dupes[0].Paths, 2)
}

func TestFindNoDuplicates(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "one.txt"), "aaa")
	writeFile(t, filepath.Join(dir, "two.txt"), "bbb")

	dupes, _, err := Find([]string{dir}, 2, zerolog.Nop())
	require.NoError(t, err)
	require.Empty(t, dupes)
}

func TestFindClampsNonPositiveWorkers(t *testing.T) {
	for _, workers := range []int{0, -1} {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "dup1.txt"), "same content")
		writeFile(t, filepath.Join(dir, "dup2.txt"), "same content")

		dupes, _, err := Find([]string{dir}, workers, zerolog.Nop())
		require.NoError(t, err)
		require.Len(t, dupes, 1)
		require.Len(t, dupes[0].Paths, 2)
	}
}

func TestFindMissingRoot(t *testing.T) {
	_, _, err := Find([]string{filepath.Join(t.TempDir(), "definitely-missing-root")}, 2, zerolog.Nop())
	require.Error(t, err)
}

func TestFindPartialRootFailureStillReturnsResults(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "dup1.txt"), "same content")
	writeFile(t, filepath.Join(dir, "dup2.txt"), "same content")

	missing := filepath.Join(t.TempDir(), "definitely-missing-root")

	dupes, _, err := Find([]string{dir, missing}, 2, zerolog.Nop())
	require.Error(t, err)
	require.Len(t, dupes, 1)
}

// TestFindSortsMultipleDuplicateGroupsByHash exercises the sort.Slice
// comparator in Find, which is only invoked when there are at least two
// duplicate sets to compare.
func TestFindSortsMultipleDuplicateGroupsByHash(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a1.txt"), "group a")
	writeFile(t, filepath.Join(dir, "a2.txt"), "group a")
	writeFile(t, filepath.Join(dir, "b1.txt"), "group b")
	writeFile(t, filepath.Join(dir, "b2.txt"), "group b")

	dupes, _, err := Find([]string{dir}, 2, zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, dupes, 2)
	require.Negative(t, bytes.Compare(dupes[0].Hash[:], dupes[1].Hash[:]))
	require.Len(t, dupes[0].Paths, 2)
}
