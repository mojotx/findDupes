package dedupe

import (
	"context"
	"crypto/sha256"
	"io"
	"os"

	"github.com/rs/zerolog/log"
)

// HashFile computes the SHA-256 hash of the file at path.
func HashFile(ctx context.Context, path string) (HashType, error) {
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
	if _, err := io.Copy(h, contextReader{ctx: ctx, reader: f}); err != nil {
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
