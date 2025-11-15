package mark

import (
	"math/rand"

	"github.com/yyyoichi/bitstream-go"
	"github.com/yyyoichi/golay"
	watermark "github.com/yyyoichi/watermark_zero"
)

var (
	DefaultShuffleSeed int64 = 1234567890
)

type (
	// EmbedMark is the interface required for embedding digital watermarks.
	// It provides bit-level access to mark data for watermark embedding operations.
	EmbedMark = watermark.EmbedMark

	// Option is a function for selecting the algorithm for mark generation.
	// It allows choosing whether to use error correction codes (ECC) and which type.
	Option      func(*markFactory)
	markFactory struct {
		encode func([]uint64, int) ([]uint64, int)
	}
)

// WithoutECC is an option that does not use error correction codes.
// It uses the mark data as-is without encoding.
func WithoutECC() Option {
	return func(mf *markFactory) {
		mf.encode = func(data []uint64, markLen int) ([]uint64, int) {
			return data, markLen
		}
	}
}

// WithGolay is an option that uses Golay code for error correction.
// seed is the seed value for shuffling the mark data.
// The generated mark is deterministically shuffled to distribute the effects
// of specific high-frequency regions in the image.
func WithGolay(seed int64) Option {
	return func(mf *markFactory) {
		mf.encode = shuffledgolay(seed).encode
	}
}

type shuffledgolay int64

func (sg shuffledgolay) encode(data []uint64, markLen int) ([]uint64, int) {
	var encoded []uint64
	enc := golay.NewEncoder(data, markLen)
	_ = enc.Encode(&encoded)
	encodedLen := enc.Bits()
	// shuffle
	r := bitstream.NewBitReader(encoded, 0, 0)
	w := bitstream.NewBitWriter[uint64](0, 0)
	seed := int64(sg)
	rd := rand.New(rand.NewSource(seed))
	rd.Shuffle(encodedLen, func(i, j int) {
		ii, _ := r.ReadBitAt(i)
		jj, _ := r.ReadBitAt(j)
		w.WriteBitAt(i, jj)
		w.WriteBitAt(j, ii)
	})
	return w.Data(), encodedLen
}
