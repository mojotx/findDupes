// Package cmd defines the findDupes cobra CLI.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/fatih/color"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/mojotx/findDupes/internal/dedupe"
	"github.com/mojotx/findDupes/internal/version"
)

var (
	verbose  bool
	workers  int
	jsonOut  bool
	statsOut bool
)

var rootCmd = &cobra.Command{
	Use:          "findDupes [flags] <directory> [directory...]",
	Short:        "Find duplicate files by content hash",
	Version:      version.Version(),
	Args:         cobra.MinimumNArgs(1),
	SilenceUsage: true,
	RunE:         runFind,
}

func init() {
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "print progress while scanning files")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", runtime.NumCPU(), "number of concurrent hashing workers")
	rootCmd.Flags().BoolVar(&jsonOut, "json", false, "print duplicate groups as newline-delimited JSON")
	rootCmd.Flags().BoolVar(&statsOut, "stats", false, "print scan statistics to stderr")
}

// Execute runs the root command.
func Execute(ctx context.Context) error {
	return rootCmd.ExecuteContext(ctx)
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

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	dupes, stats, err := dedupe.Find(ctx, args, workers, logger)
	if jsonOut {
		err = printDuplicatesJSON(os.Stdout, dupes, err)
	} else {
		printDuplicates(dupes)
	}
	if statsOut {
		printStats(os.Stderr, stats)
	}

	logger.Info().Msgf("Elapsed time: %s", time.Since(startTime))
	return err
}

type duplicateJSON struct {
	Hash  string   `json:"hash"`
	Size  int64    `json:"size"`
	Paths []string `json:"paths"`
}

func printDuplicatesJSON(w io.Writer, dupes []dedupe.DuplicateSet, scanErr error) error {
	encoder := json.NewEncoder(w)
	for _, d := range dupes {
		if err := encoder.Encode(duplicateJSON{
			Hash:  fmt.Sprintf("%x", d.Hash),
			Size:  d.Size,
			Paths: d.Paths,
		}); err != nil {
			return err
		}
	}
	return scanErr
}

func printStats(w io.Writer, stats dedupe.Stats) {
	_, _ = fmt.Fprintf(w, "files=%d skipped=%d candidates=%d duplicate_groups=%d duplicate_files=%d\n",
		stats.TotalFiles, stats.Skipped, stats.Candidates, stats.DuplicateGroups, stats.DuplicateFiles)
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
