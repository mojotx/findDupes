// Package dedupe finds duplicate files by content hash.
package dedupe

// HashType is a SHA-256 digest of a file's contents.
type HashType [32]byte

// FileEntry is a file discovered while walking the input directories.
type FileEntry struct {
	Path string
	Size int64
}

// Result is the outcome of hashing a single FileEntry.
type Result struct {
	Path string
	Size int64
	Hash HashType
}

// DuplicateSet groups the paths of files sharing the same hash.
type DuplicateSet struct {
	Hash  HashType
	Size  int64
	Paths []string
}

// Stats summarizes a Find run.
type Stats struct {
	TotalFiles int
	Skipped    int
	Candidates int
}
