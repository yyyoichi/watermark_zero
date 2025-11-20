package mark

import (
	"github.com/yyyoichi/bitstream-go"
	watermark "github.com/yyyoichi/watermark_zero"
)

var _ watermark.EmbedMark = (*Mark64)(nil)

// Mark64 is an implementation of the EmbedMark interface that manages
// embedded mark bits based on uint64.
type (
	Mark64 struct {
		markSize int
		reader   reader
	}
	reader interface {
		Bits() int
		Read8R(bits int, n int) uint8
	}
)

// New64 initializes and returns a new Mark64 instance.
// By default, it uses the Golay code with shuffle error correction algorithm.
// Custom options can be provided to change the encoding behavior.
func New64(data []uint64, markLen int, opts ...Option) *Mark64 {
	var mf markFactory
	if len(opts) == 0 {
		mf.encode = shuffledgolay(DefaultShuffleSeed).encode
	}
	for _, opt := range opts {
		opt(&mf)
	}
	var internalLen int
	data, internalLen = mf.encode(data, markLen)
	reader := bitstream.NewBitReader(data, 0, 0)
	reader.SetBits(internalLen)
	return &Mark64{
		reader:   reader,
		markSize: markLen,
	}
}

// GetBit returns the bit value at the specified position as a float64.
// The position wraps around using modulo if it exceeds the mark length.
func (m *Mark64) GetBit(at int) float64 {
	n := at % m.reader.Bits()
	return float64(m.reader.Read8R(1, n))
}

// Len returns the total number of bits in the mark.
func (m *Mark64) Len() int {
	return m.reader.Bits()
}

func (m *Mark64) ExtractSize() int {
	return m.markSize
}
