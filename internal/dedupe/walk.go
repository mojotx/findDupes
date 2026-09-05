package dedupe

import (
	"errors"
	"io/fs"
	"path/filepath"

	"github.com/rs/zerolog"
)

// WalkDirs collects all regular files under roots. Per-file walk errors are
// logged and skipped; a failing root is collected and returned as a joined error.
func WalkDirs(roots []string, logger zerolog.Logger) ([]FileEntry, error) {
	var files []FileEntry
	walker := func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			logger.Error().Err(err).Str("path", path).Msg("error walking file")
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		files = append(files, FileEntry{Path: path, Size: info.Size()})
		return nil
	}

	var errs []error
	for _, root := range roots {
		if err := filepath.Walk(root, walker); err != nil {
			errs = append(errs, err)
		}
	}
	return files, errors.Join(errs...)
}
