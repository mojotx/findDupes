# findDupes

[![CI](https://github.com/mojotx/findDupes/actions/workflows/ci.yml/badge.svg)](https://github.com/mojotx/findDupes/actions/workflows/ci.yml)
[![CodeQL](https://github.com/mojotx/findDupes/actions/workflows/codeql.yml/badge.svg)](https://github.com/mojotx/findDupes/actions/workflows/codeql.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/mojotx/findDupes.svg)](https://pkg.go.dev/github.com/mojotx/findDupes)

A command-line tool that finds duplicate files by content hash (SHA-256),
searching one or more directories concurrently.

## How it works

`findDupes` uses a two-stage process (with a size-based filter between stages) to avoid reading every file unnecessarily:

1. It recursively walks the supplied roots and collects the path and size of
	 each regular file. Reading file metadata is much faster than reading file
	 contents, so this first pass is relatively inexpensive.
2. Files whose size is unique among all discovered files are discarded because
	 they cannot have an identical copy. Files sharing a size become candidates
	 for content comparison.
3. The candidates are read concurrently by a worker pool and each file is
	 hashed with SHA-256. Files with the same hash are reported as duplicates.

The size check is only a filter: files must still have matching content hashes
to be considered duplicates. Conversely, files with different sizes are never
read or hashed against one another.

### Roots and scanning behavior

- Multiple roots are supported and are scanned as one collection of files.
- Roots are converted to absolute, symlink-resolved paths before scanning.
	This means a root supplied through a directory symlink is scanned normally.
- Repeated roots and overlapping roots, such as `dir dir` or `dir dir/sub`,
	are deduplicated before walking. The same applies when roots use different
	spellings, such as relative and absolute paths, or resolve through the same
	symlink. Each file is therefore considered only once.
- Only regular files are collected. Symlinks encountered within a scanned
	directory are not followed.
- A file that cannot be read during hashing is logged and skipped. If a root
	cannot be resolved or walked, files successfully found under other roots are
	still processed, but the command returns an error after printing any results.

Duplicate groups are printed with their SHA-256 hash, file size, and paths. The
paths within each group, and the groups themselves, are sorted for repeatable
output.

## Installation

Requires Go 1.25 or later. The [go.mod](go.mod) `go` directive is deliberately
pinned to the oldest Go version that supports the code and its dependencies,
rather than tracking the latest compiler — this keeps the requirement low for
contributors without needing a documented compatibility exception each time a
newer Go is released.

```sh
go install github.com/mojotx/findDupes@latest
```

You can also simply clone the repository and then install locally with:

```sh
go install -v ./...
```

This will install the `findDupes` binary into your `${GOBIN}` directory, e.g., `$HOME/go/bin`.

## Usage

```sh
findDupes [flags] <directory> [directory...]
```

Flags:

- `-v`, `--verbose` — print progress while scanning files
- `-w`, `--workers` — number of concurrent hashing workers (default: number of CPUs)
- `--version` — print the version number

## Known CI limitations

* The CI matrix runs the race-enabled test suite on Linux, macOS, and Windows using the Go version declared in [go.mod](go.mod). If a runner-specific test failure appears, link the current failure and address that environment directly rather than disabling macOS tests globally.
