package dedupe

import (
	"crypto/sha256"
	"io"
	"os"

	"github.com/rs/zerolog/log"
)

// HashFile computes the SHA-256 hash of the file at path.
func HashFile(path string) (HashType, error) {
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
	if _, err := io.Copy(h, f); err != nil {
		return hash, err
	}
	copy(hash[:], h.Sum(nil))
	return hash, nil
}
