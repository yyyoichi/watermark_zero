package mark

import (
	"github.com/yyyoichi/bitstream-go"
)

var (
	DefaultShuffleSeed int64 = 1234567890
)

type (
	// Option is a function for selecting the algorithm for mark generation.
	// It allows choosing whether to use error correction codes (ECC) and which type.
	Option      func(*markFactory)
	markFactory struct {
		f factroy
	}
	factroy interface {
		encode(data []uint64, markSize int) ([]uint64, int)
		decode(data []uint64, size int) *bitstream.BitReader[uint64]
		encodedLen(size int) int
	}
)

// WithoutECC is an option that does not use error correction codes.
// It uses the mark data as-is without encoding.
func WithoutECC() Option {
	return func(mf *markFactory) {
		mf.f = withoutecc{}
	}
}

// WithGolay is an option that uses Golay code for error correction.
// seed is the seed value for shuffling the mark data.
// The generated mark is deterministically shuffled to distribute the effects
// of specific high-frequency regions in the image.
func WithGolay(seed int64) Option {
	return func(mf *markFactory) {
		mf.f = shuffledgolay(seed)
	}
}
