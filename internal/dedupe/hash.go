package dedupe

import (
	"context"
	"crypto/sha256"
	"io"
	"os"

	"github.com/rs/zerolog/log"
)

const hashPrefixSize = 4 << 10

// HashFile computes the SHA-256 hash of the file at path.
func HashFile(ctx context.Context, path string) (HashType, error) {
	return hashFile(ctx, path, 0)
}

func hashPrefix(ctx context.Context, path string) (HashType, error) {
	return hashFile(ctx, path, hashPrefixSize)
}

func hashFile(ctx context.Context, path string, limit int64) (HashType, error) {
	if err := ctx.Err(); err != nil {
		return HashType{}, err
	}
	var hash HashType
	f, err := os.Open(path)
	if err != nil {
		return hash, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Error().Err(err).Msgf("failed to close file %s", path)
		}
	}()

	h := sha256.New()
	reader := io.Reader(contextReader{ctx: ctx, reader: f})
	if limit > 0 {
		reader = io.LimitReader(reader, limit)
	}
	if _, err := io.Copy(h, reader); err != nil {
		return hash, err
	}
	copy(hash[:], h.Sum(nil))
	return hash, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
