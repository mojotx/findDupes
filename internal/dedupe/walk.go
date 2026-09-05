package dedupe

import (
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/rs/zerolog"
)

// WalkDirs collects all regular files under roots. Per-file walk errors are
// logged and skipped; a failing root is collected and returned as a joined error.
// Overlapping or repeated roots (e.g. "dir dir" or "dir dir/sub") are
// deduplicated by canonical path so each file is only reported once.
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
		canonical, absErr := filepath.Abs(path)
		if absErr != nil {
			canonical = path
		}
		if _, ok := seen[canonical]; ok {
			return nil
		}
		seen[canonical] = struct{}{}
		files = append(files, FileEntry{Path: path, Size: info.Size()})
		return nil
	}

	var errs []error
	for _, r := range roots {
		root = r
		if err := filepath.Walk(root, walker); err != nil {
			errs = append(errs, err)
		}
	}
	return files, errors.Join(errs...)
}
