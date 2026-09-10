package dedupe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog"
)

func BenchmarkHashing(b *testing.B) {
	dir := b.TempDir()
	files := make([]FileEntry, 32)
	for i := range files {
		path := filepath.Join(dir, fmt.Sprintf("file-%02d.dat", i))
		data := make([]byte, 8<<20)
		for offset := range data {
			data[offset] = byte(i + offset)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			b.Fatal(err)
		}
		files[i] = FileEntry{Path: path, Size: int64(len(data))}
	}

	b.Run("plain SHA-256", func(b *testing.B) {
		for range b.N {
			for range hashAll(context.Background(), files, 4, zerolog.Nop()) {
			}
		}
	})
	b.Run("progressive", func(b *testing.B) {
		for range b.N {
			for range progressiveHashAll(context.Background(), files, 4, zerolog.Nop()) {
			}
		}
	})
}
