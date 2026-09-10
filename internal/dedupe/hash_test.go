package dedupe

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello world"), 0o600))

	h1, err := HashFile(context.Background(), path)
	require.NoError(t, err)
	h2, err := HashFile(context.Background(), path)
	require.NoError(t, err)
	require.Equal(t, h1, h2, "hash of same file differs")

	otherPath := filepath.Join(dir, "b.txt")
	require.NoError(t, os.WriteFile(otherPath, []byte("different content"), 0o600))
	h3, err := HashFile(context.Background(), otherPath)
	require.NoError(t, err)
	require.NotEqual(t, h1, h3, "hash of different files matched")
}

func TestHashFileMissing(t *testing.T) {
	_, err := HashFile(context.Background(), filepath.Join(t.TempDir(), "missing.txt"))
	require.Error(t, err)
}

// TestHashFileDirectory exercises the io.Copy error branch: reading a
// directory's contents as a file fails on both Linux and macOS.
func TestHashFileDirectory(t *testing.T) {
	_, err := HashFile(context.Background(), t.TempDir())
	require.Error(t, err)
}

func TestHashFileCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := HashFile(ctx, filepath.Join(t.TempDir(), "missing.txt"))
	require.ErrorIs(t, err, context.Canceled)
}

func TestContextReaderStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := contextReader{ctx: ctx, reader: bytes.NewReader([]byte("content"))}
	buffer := make([]byte, 1)

	count, err := reader.Read(buffer)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	cancel()
	count, err = reader.Read(buffer)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, count)
}

func TestHashPrefixOnlyReadsPrefix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.txt")
	require.NoError(t, os.WriteFile(path, append([]byte("prefix"), make([]byte, hashPrefixSize)...), 0o600))

	prefix, err := hashPrefix(context.Background(), path)
	require.NoError(t, err)
	want, err := hashFile(context.Background(), path, hashPrefixSize)
	require.NoError(t, err)
	require.Equal(t, want, prefix)
	require.NotEqual(t, prefix, mustHash(t, path))
}

func mustHash(t *testing.T, path string) HashType {
	t.Helper()
	hash, err := HashFile(context.Background(), path)
	require.NoError(t, err)
	return hash
}
