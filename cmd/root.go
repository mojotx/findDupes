// Package cmd defines the findDupes cobra CLI.
package cmd

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/mojotx/findDupes/internal/dedupe"
)

var (
	verbose bool
	workers int
)

var rootCmd = &cobra.Command{
	Use:          "findDupes [flags] <directory> [directory...]",
	Short:        "Find duplicate files by content hash",
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE:         runFind,
}

func init() {
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print progress while scanning files")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", runtime.NumCPU(), "number of concurrent hashing workers")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func runFind(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	logLevel := zerolog.InfoLevel
	if verbose {
		logLevel = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(logLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	})
	logger := log.Logger

	dupes, _, err := dedupe.Find(args, workers, logger)
	if err != nil {
		return err
	}

	printDuplicates(dupes)

	logger.Info().Msgf("Elapsed time: %s", time.Since(startTime))
	return nil
}

func printDuplicates(dupes []dedupe.DuplicateSet) {
	fmt.Println("")
	black := color.New(color.FgHiBlack)
	for _, d := range dupes {
		_, _ = black.Printf("%x: %d (%d)\n", d.Hash, d.Size, len(d.Paths))
		for _, path := range d.Paths {
			fmt.Printf("%q\n", path)
		}
		fmt.Println("")
	}
}
