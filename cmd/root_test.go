package cmd

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/mojotx/findDupes/internal/dedupe"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

// captureStdout redirects os.Stdout for the duration of f and returns
// whatever was written to it.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() {
		os.Stdout = old
		_ = r.Close()
		_ = w.Close()
	}()
	f()
	require.NoError(t, w.Close())

	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func captureStderr(t *testing.T, f func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() {
		os.Stderr = old
		_ = r.Close()
		_ = w.Close()
	}()
	f()
	require.NoError(t, w.Close())

	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func TestPrintDuplicates(t *testing.T) {
	dupes := []dedupe.DuplicateSet{
		{
			Hash:  dedupe.HashType{0xde, 0xad, 0xbe, 0xef},
			Size:  42,
			Paths: []string{"/tmp/a.txt", "/tmp/b.txt"},
		},
	}

	out := captureStdout(t, func() {
		printDuplicates(dupes)
	})

	// The hash/size line goes through fatih/color, which caches os.Stdout at
	// package init time and so isn't captured by swapping os.Stdout here;
	// only the plain fmt.Println/Printf paths are observable in out.
	require.Contains(t, out, `"/tmp/a.txt"`)
	require.Contains(t, out, `"/tmp/b.txt"`)
}

func TestPrintDuplicatesEmpty(t *testing.T) {
	out := captureStdout(t, func() {
		printDuplicates(nil)
	})
	require.Equal(t, "\n", out)
}

func TestRunFind(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dup1.txt"), []byte("same"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dup2.txt"), []byte("same"), 0o600))

	out := captureStdout(t, func() {
		err := runFind(rootCmd, []string{dir})
		require.NoError(t, err)
	})
	require.Contains(t, out, "dup1.txt")
	require.Contains(t, out, "dup2.txt")
}

func TestRunFindMissingRoot(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	_ = captureStdout(t, func() {
		err := runFind(rootCmd, []string{missing})
		require.Error(t, err)
	})
}

func TestRunFindVerbose(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("aaa"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("aaa"), 0o600))

	origVerbose := verbose
	origLogger := log.Logger
	origLevel := zerolog.GlobalLevel()
	verbose = true
	t.Cleanup(func() {
		verbose = origVerbose
		log.Logger = origLogger
		zerolog.SetGlobalLevel(origLevel)
	})

	stderr := captureStderr(t, func() {
		err := runFind(rootCmd, []string{dir})
		require.NoError(t, err)
	})
	require.Contains(t, stderr, "scanning file")
}

func TestExecute(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("aaa"), 0o600))

	rootCmd.SetArgs([]string{dir})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	_ = captureStdout(t, func() {
		require.NoError(t, Execute())
	})
}
