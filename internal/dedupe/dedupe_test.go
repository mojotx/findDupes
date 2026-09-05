package dedupe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

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

func TestFindMissingRoot(t *testing.T) {
	_, _, err := Find([]string{filepath.Join(os.TempDir(), "definitely-missing-root")}, 2, zerolog.Nop())
	require.Error(t, err)
}
