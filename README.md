# findDupes

[![CI](https://github.com/mojotx/findDupes/actions/workflows/ci.yml/badge.svg)](https://github.com/mojotx/findDupes/actions/workflows/ci.yml)
[![CodeQL](https://github.com/mojotx/findDupes/actions/workflows/codeql.yml/badge.svg)](https://github.com/mojotx/findDupes/actions/workflows/codeql.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/mojotx/findDupes.svg)](https://pkg.go.dev/github.com/mojotx/findDupes)

A command-line tool that finds duplicate files by content hash (SHA-256),
searching one or more directories concurrently.

## How it works

`findDupes` filters candidates before hashing and uses progressive hashing to
avoid reading every file unnecessarily:

1. It recursively walks the supplied roots and collects the path and size of
   each regular file. Reading file metadata is much faster than reading file
   contents, so this first pass is relatively inexpensive.
2. Files whose size is unique among all discovered files are discarded because
	 they cannot have an identical copy. Files sharing a size become candidates
	 for content comparison.
3. A worker pool hashes the first 4 KiB of each candidate. Only files sharing
	 that prefix hash are read again and hashed in full with SHA-256.
4. Files with the same full hash are reported as duplicates. Hard-linked paths
	 to the same underlying file are collected only once.

The size and prefix checks are only filters: files must still have matching
full content hashes to be considered duplicates. Conversely, files with
different sizes or prefix hashes are never fully hashed against one another.

### Roots and scanning behavior

- Multiple roots are supported and are scanned as one collection of files.
- Roots are converted to absolute, symlink-resolved paths before scanning.
	This means a root supplied through a directory symlink is scanned normally.
- Repeated roots and overlapping roots, such as `dir dir` or `dir dir/sub`,
	are deduplicated before walking. The same applies when roots use different
	spellings, such as relative and absolute paths, or resolve through the same
	 symlink. Filesystem identities are tracked with constant-time lookups, so
	 each file is considered only once without comparing it to every prior file.
- Only regular files are collected. Symlinks encountered within a scanned
	directory are not followed.
- Multiple paths to the same hard-linked file are collected once.
- A file that cannot be read during hashing is logged and skipped. If a root
	cannot be resolved or walked, files successfully found under other roots are
	still processed, but the command returns an error after printing any results.
- `Ctrl-C` and `SIGTERM` cancel an active scan. Work stops between filesystem
	 reads and the command returns the cancellation error.

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
- `--json` — print duplicate groups as newline-delimited JSON
- `--stats` — print scan statistics to stderr
- `--version` — print the version number

JSON output has one object per duplicate group with `hash`, `size`, and
`paths` fields. Statistics include discovered files, size-filtered files,
content-hash candidates, duplicate groups, and duplicate files. Statistics
are written to stderr so JSON output remains safe to pipe into tools such as
`jq`.

For example, to inspect duplicate groups with `jq` while keeping statistics
on the terminal:

```sh
findDupes --json --stats ~/Downloads > duplicates.ndjson
jq 'select(.size > 1048576)' duplicates.ndjson
```

## Known CI limitations

* The CI matrix runs the race-enabled test suite on Linux, macOS, and Windows using the Go version declared in [go.mod](go.mod). If a runner-specific test failure appears, link the current failure and address that environment directly rather than disabling macOS tests globally.
