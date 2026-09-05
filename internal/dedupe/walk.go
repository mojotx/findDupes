package dedupe

import (
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/rs/zerolog"
)

// WalkDirs collects all regular files under roots. Per-file walk errors are
// logged and skipped; a failing root is collected and returned as a joined error.
// Overlapping or repeated roots (e.g. "dir dir" or "dir dir/sub"), and roots
// that are themselves directory symlinks or symlinked via an intermediate
// component, are deduplicated by resolving each root to its canonical path
// once before walking it. filepath.Walk never follows symlinks it encounters
// during the walk itself, so resolving only the root is sufficient: every
// path it reports is already rooted at that canonical, symlink-free prefix.
func WalkDirs(roots []string, logger zerolog.Logger) ([]FileEntry, error) {
	var files []FileEntry
	seen := make(map[string]struct{})
	var root string
	walker := func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			logger.Error().Err(err).Str("path", path).Msg("error walking file")
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if _, ok := seen[path]; ok {
			return nil
		}
		seen[path] = struct{}{}
		files = append(files, FileEntry{Path: path, Size: info.Size()})
		return nil
	}

	var errs []error
	for _, r := range roots {
		root = canonicalRoot(r)
		if err := filepath.Walk(root, walker); err != nil {
			errs = append(errs, err)
		}
	}
	return files, errors.Join(errs...)
}

// canonicalRoot resolves root to an absolute, symlink-free path so that
// equivalent roots (relative vs. absolute, or reached via a symlink) share a
// single spelling, and so that a root which is itself a directory symlink is
// actually walked instead of being skipped as non-regular. If resolution
// fails (e.g. a missing root), it falls back to the best-effort form so
// filepath.Walk still reports its own, more specific error.
func canonicalRoot(root string) string {
	abs, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return resolved
}
