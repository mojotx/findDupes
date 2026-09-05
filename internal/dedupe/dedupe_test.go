package dedupe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func TestFilterBySize(t *testing.T) {
	files := []FileEntry{
		{Path: "a", Size: 10},
		{Path: "b", Size: 10},
		{Path: "c", Size: 20},
	}
	candidates, skipped := FilterBySize(files)
	if skipped != 1 {
		t.Fatalf("expected 1 skipped, got %d", skipped)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d: %+v", len(candidates), candidates)
	}
}

func TestFind(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "dup1.txt"), "same content")
	writeFile(t, filepath.Join(dir, "dup2.txt"), "same content")
	writeFile(t, filepath.Join(dir, "unique.txt"), "one of a kind")

	dupes, stats, err := Find([]string{dir}, 2, zerolog.Nop())
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if stats.TotalFiles != 3 {
		t.Fatalf("expected 3 total files, got %d", stats.TotalFiles)
	}
	if stats.Skipped != 1 {
		t.Fatalf("expected 1 skipped, got %d", stats.Skipped)
	}
	if len(dupes) != 1 {
		t.Fatalf("expected 1 duplicate set, got %d: %+v", len(dupes), dupes)
	}
	if len(dupes[0].Paths) != 2 {
		t.Fatalf("expected 2 paths in duplicate set, got %d: %+v", len(dupes[0].Paths), dupes[0].Paths)
	}
}

func TestFindNoDuplicates(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "one.txt"), "aaa")
	writeFile(t, filepath.Join(dir, "two.txt"), "bbb")

	dupes, _, err := Find([]string{dir}, 2, zerolog.Nop())
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(dupes) != 0 {
		t.Fatalf("expected no duplicate sets, got %d: %+v", len(dupes), dupes)
	}
}

func TestFindMissingRoot(t *testing.T) {
	dupes, stats, err := Find([]string{filepath.Join(os.TempDir(), "definitely-missing-root")}, 2, zerolog.Nop())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(dupes) != 0 || stats.TotalFiles != 0 {
		t.Fatalf("expected no files or duplicates, got dupes=%+v stats=%+v", dupes, stats)
	}
}
