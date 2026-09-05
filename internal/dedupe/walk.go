package dedupe

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rs/zerolog"
)

// WalkDirs collects all regular files under roots. Per-file walk errors are
// logged and skipped; a failing root (including one that cannot be resolved,
// such as a dangling symlink) is collected and returned as a joined error.
// Overlapping or repeated roots (e.g. "dir dir" or "dir dir/sub"), and roots
// that are themselves directory symlinks or symlinked via an intermediate
// component, are resolved to a canonical path and deduplicated before any
// walk starts, so each distinct subtree is only ever walked once.
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
	canonical := make([]string, 0, len(roots))
	for _, r := range roots {
		c, err := canonicalRoot(r)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		canonical = append(canonical, c)
	}

	for _, r := range dedupeContainedRoots(canonical) {
		root = r
		if err := filepath.Walk(root, walker); err != nil {
			errs = append(errs, err)
		}
	}
	return files, errors.Join(errs...)
}

// dedupeContainedRoots removes exact duplicates and any root nested inside
// another root in the list, so filepath.Walk is only invoked once per
// distinct subtree even when overlapping or repeated roots are supplied.
// roots must already be canonical (absolute, symlink-resolved) paths.
func dedupeContainedRoots(roots []string) []string {
	sorted := append([]string(nil), roots...)
	sort.Strings(sorted)

	result := make([]string, 0, len(sorted))
	for _, r := range sorted {
		if len(result) > 0 {
			container := result[len(result)-1]
			if r == container || strings.HasPrefix(r, container+string(filepath.Separator)) {
				continue
			}
		}
		result = append(result, r)
	}
	return result
}

// canonicalRoot resolves root to an absolute, symlink-free path so that
// equivalent roots (relative vs. absolute, or reached via a symlink) share a
// single spelling, and so that a root which is itself a directory symlink is
// actually walked instead of being skipped as non-regular. Resolution
// failures (a missing root, a dangling symlink, etc.) are returned as an
// error rather than silently falling back to an unresolved path: filepath.Walk
// would otherwise Lstat that path directly, and for a dangling symlink it
// would skip it as non-regular and report success with zero files.
func canonicalRoot(root string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return resolved, nil
}
