package main

import (
	"crypto/sha256"
	"fmt"
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
	log.Level(zerolog.InfoLevel)
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
			return err
		}

		if !info.IsDir() {

			var data []byte

			data, err = os.ReadFile(path)
			if err != nil {
				log.Error().Err(err).Str("path", path).Msg("error calling os.ReadFile")
				return err
			}

			hash := sha256.Sum256(data)
			log.Debug().Str("path", path).Int64("size", info.Size()).Msgf("%x", hash)
			hashMap[hash] = append(hashMap[hash], path)
			if _, found := sizeMap[hash]; !found {
				sizeMap[hash] = info.Size()
			}
		}
		return nil
	}

	for _, arg := range os.Args[1:] {
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
			_, _ = black.Printf("%x: %d (%d)\n", hash, size, len(pathSlice)+1)

			for _, fileName := range pathSlice {
				fmt.Printf("%q\n", fileName)
			}
			fmt.Println("")
		}
	}
}
