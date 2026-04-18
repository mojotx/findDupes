package main

import (
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type HashType [32]byte

func main() {

	verbose := flag.Bool("verbose", false, "print progress while scanning files")
	flag.BoolVar(verbose, "v", false, "print progress while scanning files (shorthand)")
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

	hashMap := make(map[HashType][]string)
	sizeMap := make(map[HashType]int64)

	var walker filepath.WalkFunc
	walker = func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("error walking file")
			return nil
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		log.Debug().Str("path", path).Int64("size", info.Size()).Msg("scanning file")

		f, err := os.Open(path)
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("error opening file")
			return nil
		}

		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			log.Error().Err(err).Str("path", path).Msg("error hashing file")
			return nil
		}
		f.Close()

		var hash HashType
		copy(hash[:], h.Sum(nil))
		log.Debug().Str("path", path).Int64("size", info.Size()).Msgf("%x", hash)
		hashMap[hash] = append(hashMap[hash], path)
		if _, found := sizeMap[hash]; !found {
			sizeMap[hash] = info.Size()
		}
		return nil
	}

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	for _, arg := range args {
		if err := filepath.Walk(arg, walker); err != nil {
			log.Error().Err(err).Str("arg", arg).Msg("top level")
		}
	}

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
