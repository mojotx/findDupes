package dedupe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func TestWalkDirsCollectsRegularFiles(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("Mkdir: %v", err)
	}
	writeFile(t, filepath.Join(dir, "a.txt"), "aaa")
	writeFile(t, filepath.Join(sub, "b.txt"), "bb")

	files, err := WalkDirs([]string{dir}, zerolog.Nop())
	if err != nil {
		t.Fatalf("WalkDirs: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %+v", len(files), files)
	}
}

func TestWalkDirsMissingRootIsLoggedNotReturned(t *testing.T) {
	// filepath.Walk invokes the walk func with the lstat error, which we log
	// and swallow (return nil), matching the original main.go behavior.
	files, err := WalkDirs([]string{filepath.Join(t.TempDir(), "does-not-exist")}, zerolog.Nop())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no files, got %+v", files)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}
