package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	})

	var walker filepath.WalkFunc
	walker = func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Error().Err(err).Str("path", path).Msg("error walking file")
			return err
		}
		log.Info().Str("path", path).Msg("howdy")
		return nil
	}

	for _, arg := range os.Args[1:] {
		if err := filepath.Walk(arg, walker); err != nil {
			log.Error().Err(err).Str("arg", arg).Msg("top level")
		}
	}
}
