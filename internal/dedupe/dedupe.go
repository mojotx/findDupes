package dedupe

import (
	"bytes"
	"context"
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
	candidateCount := 0
	for _, count := range sizeCount {
		if count > 1 {
			candidateCount += count
		}
	}
	candidates = make([]FileEntry, 0, candidateCount)
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
// If some roots fail to walk, files successfully collected from the others are
// still processed; the walk error is returned alongside whatever results were found.
// Cancellation stops the scan and returns the context error.
func Find(ctx context.Context, roots []string, workers int, logger zerolog.Logger) ([]DuplicateSet, Stats, error) {
	if err := ctx.Err(); err != nil {
		return nil, Stats{}, err
	}
	files, walkErr := WalkDirs(ctx, roots, logger)
	if err := ctx.Err(); err != nil {
		return nil, Stats{}, err
	}
	if walkErr != nil {
		logger.Error().Err(walkErr).Msg("one or more roots could not be fully scanned")
	}

	candidates, skipped := FilterBySize(files)
	stats := Stats{
		TotalFiles: len(files),
		Skipped:    skipped,
		Candidates: len(candidates),
	}
	if workers < 1 {
		workers = 1
	}
	logger.Info().
		Int("total_files", stats.TotalFiles).
		Int("skipped_unique_size", stats.Skipped).
		Int("candidates", stats.Candidates).
		Int("workers", workers).
		Msg("starting concurrent hashing")

	hashMap := make(map[HashType][]string)
	sizeMap := make(map[HashType]int64)
	for r := range progressiveHashAll(ctx, candidates, workers, logger) {
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
			stats.DuplicateGroups++
			stats.DuplicateFiles += len(paths)
		}
	}
	sort.Slice(dupes, func(i, j int) bool {
		return bytes.Compare(dupes[i].Hash[:], dupes[j].Hash[:]) < 0
	})

	if err := ctx.Err(); err != nil {
		return dupes, stats, err
	}
	return dupes, stats, walkErr
}

// hashAll hashes candidates concurrently using a fixed-size worker pool,
// streaming results back to the caller instead of buffering them all in memory.
// Cancellation stops workers and closes the result channel.
func hashAll(ctx context.Context, candidates []FileEntry, workers int, logger zerolog.Logger) <-chan Result {
	return hashAllWith(ctx, candidates, workers, logger, HashFile)
}

func progressiveHashAll(ctx context.Context, candidates []FileEntry, workers int, logger zerolog.Logger) <-chan Result {
	results := make(chan Result, workers)
	go func() {
		defer close(results)
		groups := make(map[HashType][]FileEntry)
		for result := range hashAllWith(ctx, candidates, workers, logger, hashPrefix) {
			groups[result.Hash] = append(groups[result.Hash], FileEntry{Path: result.Path, Size: result.Size})
		}
		var collisions []FileEntry
		for _, group := range groups {
			if len(group) > 1 {
				collisions = append(collisions, group...)
			}
		}
		for result := range hashAll(ctx, collisions, workers, logger) {
			select {
			case results <- result:
			case <-ctx.Done():
				return
			}
		}
	}()
	return results
}

func hashAllWith(ctx context.Context, candidates []FileEntry, workers int, logger zerolog.Logger, hashFile func(context.Context, string) (HashType, error)) <-chan Result {
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
			for {
				var fe FileEntry
				var ok bool
				select {
				case <-ctx.Done():
					return
				case fe, ok = <-jobs:
					if !ok {
						return
					}
				}
				logger.Debug().Str("path", fe.Path).Int64("size", fe.Size).Msg("scanning file")
				hash, err := hashFile(ctx, fe.Path)
				if err != nil {
					logger.Error().Err(err).Str("path", fe.Path).Msg("error hashing file")
					continue
				}
				logger.Debug().Str("path", fe.Path).Int64("size", fe.Size).Msgf("%x", hash)
				select {
				case results <- Result{Path: fe.Path, Size: fe.Size, Hash: hash}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, fe := range candidates {
			select {
			case jobs <- fe:
			case <-ctx.Done():
				close(jobs)
				return
			}
		}
		close(jobs)
	}()

	return results
}
