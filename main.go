package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type HashType [32]byte

type fileEntry struct {
	path string
	size int64
}

type hashResult struct {
	path string
	size int64
	hash HashType
}

func hashFile(path string) (HashType, error) {
	var hash HashType
	f, err := os.Open(path)
	if err != nil {
		return hash, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return hash, err
	}
	copy(hash[:], h.Sum(nil))
	return hash, nil
}

func main() {

	verbose := flag.Bool("verbose", false, "print progress while scanning files")
	flag.BoolVar(verbose, "v", false, "print progress while scanning files (shorthand)")
	workers := flag.Int("workers", runtime.NumCPU(), "number of concurrent hashing workers")
	flag.IntVar(workers, "w", runtime.NumCPU(), "number of concurrent hashing workers (shorthand)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <directory> [directory...]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	startTime := time.Now()

	logLevel := zerolog.InfoLevel
	if *verbose {
		logLevel = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(logLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	})

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Phase 1: Walk directories and collect file paths
	var files []fileEntry
	walker := func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("error walking file")
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		files = append(files, fileEntry{path: path, size: info.Size()})
		return nil
	}

	for _, arg := range args {
		if err := filepath.Walk(arg, walker); err != nil {
			log.Error().Err(err).Str("arg", arg).Msg("top level")
		}
	}

	log.Info().Int("files", len(files)).Int("workers", *workers).Msg("starting concurrent hashing")

	// Phase 2: Hash files concurrently using a worker pool
	jobs := make(chan fileEntry, *workers)
	results := make(chan hashResult, *workers)

	var wg sync.WaitGroup
	for range *workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for fe := range jobs {
				log.Debug().Str("path", fe.path).Int64("size", fe.size).Msg("scanning file")
				hash, err := hashFile(fe.path)
				if err != nil {
					log.Error().Err(err).Str("path", fe.path).Msg("error hashing file")
					continue
				}
				log.Debug().Str("path", fe.path).Int64("size", fe.size).Msgf("%x", hash)
				results <- hashResult{path: fe.path, size: fe.size, hash: hash}
			}
		}()
	}

	// Close results channel once all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Send jobs to workers
	go func() {
		for _, fe := range files {
			jobs <- fe
		}
		close(jobs)
	}()

	// Phase 3: Collect results
	hashMap := make(map[HashType][]string)
	sizeMap := make(map[HashType]int64)
	for r := range results {
		hashMap[r.hash] = append(hashMap[r.hash], r.path)
		if _, found := sizeMap[r.hash]; !found {
			sizeMap[r.hash] = r.size
		}
	}

	// Phase 4: Print duplicates
	fmt.Println("")

	black := color.New(color.FgHiBlack)

	for hash := range hashMap {
		pathSlice := hashMap[hash]
		size := sizeMap[hash]
		if len(pathSlice) > 1 {
			_, _ = black.Printf("%x: %d (%d)\n", hash, size, len(pathSlice))

			for _, fileName := range pathSlice {
				fmt.Printf("%q\n", fileName)
			}
			fmt.Println("")
		}
	}

	elapsedTime := time.Since(startTime)
	log.Info().Msgf("Elapsed time: %s", elapsedTime)
}
