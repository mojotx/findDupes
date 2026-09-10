package dedupe

import (
	"bytes"
	"context"
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
	for r := range hashAll(context.Background(), candidates, 0, zerolog.Nop()) {
		results = append(results, r)
	}
	require.Len(t, results, 1)
}

func TestHashAllSkipsUnhashableFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.txt")
	candidates := []FileEntry{{Path: missing, Size: 3}}

	var results []Result
	for r := range hashAll(context.Background(), candidates, 2, zerolog.Nop()) {
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

	dupes, stats, err := Find(context.Background(), []string{dir}, 2, zerolog.Nop())
	require.NoError(t, err)
	require.Equal(t, 3, stats.TotalFiles)
	require.Equal(t, 1, stats.Skipped)
	require.Equal(t, 1, stats.DuplicateGroups)
	require.Equal(t, 2, stats.DuplicateFiles)
	require.Len(t, dupes, 1)
	require.Len(t, dupes[0].Paths, 2)
}

func TestFindCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dupes, stats, err := Find(ctx, []string{t.TempDir()}, 2, zerolog.Nop())
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, dupes)
	require.Equal(t, Stats{}, stats)
}

func TestFindNoDuplicates(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "one.txt"), "aaa")
	writeFile(t, filepath.Join(dir, "two.txt"), "bbb")

	dupes, _, err := Find(context.Background(), []string{dir}, 2, zerolog.Nop())
	require.NoError(t, err)
	require.Empty(t, dupes)
}

func TestFindProgressiveHashConfirmsFullContent(t *testing.T) {
	dir := t.TempDir()
	prefix := bytes.Repeat([]byte("p"), hashPrefixSize)
	writeFile(t, filepath.Join(dir, "different-a.txt"), string(append(append([]byte{}, prefix...), bytes.Repeat([]byte("a"), 32)...)))
	writeFile(t, filepath.Join(dir, "different-b.txt"), string(append(append([]byte{}, prefix...), bytes.Repeat([]byte("b"), 32)...)))
	writeFile(t, filepath.Join(dir, "same-a.txt"), string(append(append([]byte{}, prefix...), bytes.Repeat([]byte("c"), 32)...)))
	writeFile(t, filepath.Join(dir, "same-b.txt"), string(append(append([]byte{}, prefix...), bytes.Repeat([]byte("c"), 32)...)))

	dupes, _, err := Find(context.Background(), []string{dir}, 2, zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, dupes, 1)
	canonicalDir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		filepath.Join(canonicalDir, "same-a.txt"),
		filepath.Join(canonicalDir, "same-b.txt"),
	}, dupes[0].Paths)
}

func TestFindClampsNonPositiveWorkers(t *testing.T) {
	for _, workers := range []int{0, -1} {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "dup1.txt"), "same content")
		writeFile(t, filepath.Join(dir, "dup2.txt"), "same content")

		dupes, _, err := Find(context.Background(), []string{dir}, workers, zerolog.Nop())
		require.NoError(t, err)
		require.Len(t, dupes, 1)
		require.Len(t, dupes[0].Paths, 2)
	}
}

func TestFindMissingRoot(t *testing.T) {
	_, _, err := Find(context.Background(), []string{filepath.Join(t.TempDir(), "definitely-missing-root")}, 2, zerolog.Nop())
	require.Error(t, err)
}

func TestFindPartialRootFailureStillReturnsResults(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "dup1.txt"), "same content")
	writeFile(t, filepath.Join(dir, "dup2.txt"), "same content")

	missing := filepath.Join(t.TempDir(), "definitely-missing-root")

	dupes, _, err := Find(context.Background(), []string{dir, missing}, 2, zerolog.Nop())
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

	dupes, _, err := Find(context.Background(), []string{dir}, 2, zerolog.Nop())
	require.NoError(t, err)
	require.Len(t, dupes, 2)
	for groupIndex := 1; groupIndex < len(dupes); groupIndex++ {
		require.Less(t, bytes.Compare(dupes[groupIndex-1].Hash[:], dupes[groupIndex].Hash[:]), 0)
	}
	require.Len(t, dupes[0].Paths, 2)
}
