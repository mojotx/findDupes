package dedupe

import (
	"errors"
	"io/fs"
	"path/filepath"
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
//
// Containment is checked with filepath.Rel against every other retained root,
// rather than lexical sorting plus a string-prefix/adjacency check: sorting
// does not guarantee a root's ancestor ends up adjacent to it (e.g. ".../a",
// ".../a!", ".../a/sub" sort in that order, separating "a" from its
// descendant "a/sub"), and a raw prefix check on "root+separator" mishandles
// a filesystem root like "/" (producing a doubled separator). filepath.Rel is
// component-aware, so both problems are avoided.
func dedupeContainedRoots(roots []string) []string {
	unique := make([]string, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))
	for _, r := range roots {
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		unique = append(unique, r)
	}

	result := make([]string, 0, len(unique))
	for i, r := range unique {
		contained := false
		for j, other := range unique {
			if i != j && isWithinRoot(r, other) {
				contained = true
				break
			}
		}
		if !contained {
			result = append(result, r)
		}
	}
	return result
}

// isWithinRoot reports whether path is root itself or a descendant of it.
// Both must already be canonical (absolute, symlink-resolved) paths.
func isWithinRoot(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
