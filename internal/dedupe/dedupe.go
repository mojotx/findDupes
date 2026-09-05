package dedupe

import (
	"bytes"
	"sort"
	"sync"

	"github.com/rs/zerolog"
)

// FilterBySize keeps only files whose size matches at least one other file,
// since a file with a unique size cannot be a duplicate of anything.
func FilterBySize(files []FileEntry) (candidates []FileEntry, skipped int) {
	sizeCount := make(map[int64]int, len(files))
	for _, fe := range files {
		sizeCount[fe.Size]++
	}
	candidates = make([]FileEntry, 0, len(files))
	for _, fe := range files {
		if sizeCount[fe.Size] > 1 {
			candidates = append(candidates, fe)
		} else {
			skipped++
		}
	}
	return candidates, skipped
}

// Find walks roots, hashes candidate files concurrently across workers
// goroutines, and returns duplicate sets sorted by hash for deterministic output.
func Find(roots []string, workers int, logger zerolog.Logger) ([]DuplicateSet, Stats, error) {
	files, err := WalkDirs(roots, logger)
	if err != nil {
		return nil, Stats{}, err
	}

	candidates, skipped := FilterBySize(files)
	stats := Stats{
		TotalFiles: len(files),
		Skipped:    skipped,
		Candidates: len(candidates),
	}
	logger.Info().
		Int("total_files", stats.TotalFiles).
		Int("skipped_unique_size", stats.Skipped).
		Int("candidates", stats.Candidates).
		Int("workers", workers).
		Msg("starting concurrent hashing")

	hashMap := make(map[HashType][]string)
	sizeMap := make(map[HashType]int64)
	for r := range hashAll(candidates, workers, logger) {
		hashMap[r.Hash] = append(hashMap[r.Hash], r.Path)
		if _, found := sizeMap[r.Hash]; !found {
			sizeMap[r.Hash] = r.Size
		}
	}

	var dupes []DuplicateSet
	for hash, paths := range hashMap {
		if len(paths) > 1 {
			sort.Strings(paths)
			dupes = append(dupes, DuplicateSet{Hash: hash, Size: sizeMap[hash], Paths: paths})
		}
	}
	sort.Slice(dupes, func(i, j int) bool {
		return bytes.Compare(dupes[i].Hash[:], dupes[j].Hash[:]) < 0
	})

	return dupes, stats, nil
}

// hashAll hashes candidates concurrently using a fixed-size worker pool,
// streaming results back to the caller instead of buffering them all in memory.
func hashAll(candidates []FileEntry, workers int, logger zerolog.Logger) <-chan Result {
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan FileEntry, workers)
	results := make(chan Result, workers)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fe := range jobs {
				logger.Debug().Str("path", fe.Path).Int64("size", fe.Size).Msg("scanning file")
				hash, err := HashFile(fe.Path)
				if err != nil {
					logger.Error().Err(err).Str("path", fe.Path).Msg("error hashing file")
					continue
				}
				logger.Debug().Str("path", fe.Path).Int64("size", fe.Size).Msgf("%x", hash)
				results <- Result{Path: fe.Path, Size: fe.Size, Hash: hash}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, fe := range candidates {
			jobs <- fe
		}
		close(jobs)
	}()

	return results
}
