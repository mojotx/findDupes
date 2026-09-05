package dedupe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hello world"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	h1, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}
	h2, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("hash of same file differs: %x vs %x", h1, h2)
	}

	otherPath := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(otherPath, []byte("different content"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	h3, err := HashFile(otherPath)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}
	if h1 == h3 {
		t.Fatalf("hash of different files matched: %x", h1)
	}
}

func TestHashFileMissing(t *testing.T) {
	if _, err := HashFile(filepath.Join(t.TempDir(), "missing.txt")); err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
