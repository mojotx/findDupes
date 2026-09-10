package dedupe

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rs/zerolog"
)

// WalkDirs collects regular files under roots. Per-file errors are logged and
// skipped; root errors are returned. Overlapping roots are walked once.
func WalkDirs(roots []string, logger zerolog.Logger) ([]FileEntry, error) {
	var files []FileEntry
	seen := make(map[string]struct{})
	var root string
	walker := func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			logger.Error().Err(err).Str("path", path).Msg("error walking file")
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
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
		if err := filepath.WalkDir(root, walker); err != nil {
			errs = append(errs, err)
		}
	}
	return files, errors.Join(errs...)
}

// dedupeContainedRoots removes duplicate and nested roots. Roots must already
// be absolute and symlink-resolved; containment uses filesystem identity.
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

	cache := make(identityCache, len(unique))
	result := make([]string, 0, len(unique))
	for i, r := range unique {
		contained := false
		for j, other := range unique {
			if i == j || !cache.isWithinRoot(r, other) {
				continue
			}
			if cache.isWithinRoot(other, r) {
				// r and other are within each other, meaning they denote the
				// same directory on disk despite being spelled differently
				// (e.g. case variants on a case-insensitive filesystem). Keep
				// only the first occurrence instead of dropping both.
				if j < i {
					contained = true
				}
				continue
			}
			contained = true
			break
		}
		if !contained {
			result = append(result, r)
		}
	}
	return result
}

// identityCache caches filesystem identity lookups by path.
type identityCache map[string]os.FileInfo

// stat returns cached file information and reports whether the lookup worked.
func (c identityCache) stat(path string) (os.FileInfo, bool) {
	if info, ok := c[path]; ok {
		return info, info != nil
	}
	info, err := os.Stat(path)
	if err != nil {
		c[path] = nil
		return nil, false
	}
	c[path] = info
	return info, true
}

// isWithinRoot reports whether path is root or one of its descendants, using
// filesystem identity rather than path strings.
func (c identityCache) isWithinRoot(path, root string) bool {
	rootInfo, ok := c.stat(root)
	if !ok {
		return false
	}
	for {
		if info, ok := c.stat(path); ok && os.SameFile(info, rootInfo) {
			return true
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false
		}
		path = parent
	}
}

// canonicalRoot returns an absolute, symlink-free root. Resolution failures
// are returned so missing and dangling roots do not appear empty.
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
