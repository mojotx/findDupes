package dedupe

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello world"), 0o600))

	h1, err := HashFile(path)
	require.NoError(t, err)
	h2, err := HashFile(path)
	require.NoError(t, err)
	require.Equal(t, h1, h2, "hash of same file differs")

	otherPath := filepath.Join(dir, "b.txt")
	require.NoError(t, os.WriteFile(otherPath, []byte("different content"), 0o600))
	h3, err := HashFile(otherPath)
	require.NoError(t, err)
	require.NotEqual(t, h1, h3, "hash of different files matched")
}

func TestHashFileMissing(t *testing.T) {
	_, err := HashFile(filepath.Join(t.TempDir(), "missing.txt"))
	require.Error(t, err)
}
