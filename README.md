# findDupes

[![CI](https://github.com/mojotx/findDupes/actions/workflows/ci.yml/badge.svg)](https://github.com/mojotx/findDupes/actions/workflows/ci.yml)
[![CodeQL](https://github.com/mojotx/findDupes/actions/workflows/codeql.yml/badge.svg)](https://github.com/mojotx/findDupes/actions/workflows/codeql.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/mojotx/findDupes.svg)](https://pkg.go.dev/github.com/mojotx/findDupes)

A command-line tool that finds duplicate files by content hash (SHA-256),
searching one or more directories concurrently.

## Installation

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
