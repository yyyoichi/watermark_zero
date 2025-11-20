package mark

import (
	"github.com/yyyoichi/bitstream-go"
	"github.com/yyyoichi/golay"
	watermark "github.com/yyyoichi/watermark_zero"
)

var _ watermark.MarkCore = (*Mark64)(nil)
var _ watermark.EmbedMark = (*Mark64)(nil)
var _ watermark.ExtractMark = (*Mark64)(nil)
var _ watermark.MarkDecoder = (*Mark64)(nil)

// Mark64 is an implementation of the EmbedMark interface that manages
// embedded mark bits based on uint64.
type Mark64 struct {
	size   int
	reader *bitstream.BitReader[uint64]
	mf     markFactory
}

// New64 initializes and returns a new Mark64 instance.
// By default, it uses the Golay code with shuffle error correction algorithm.
// Custom options can be provided to change the encoding behavior.
func New64(data []uint64, size int, opts ...Option) *Mark64 {
	if len(opts) == 0 {
		opts = append(opts, WithGolay(DefaultShuffleSeed))
	}
	var mf markFactory
	for _, opt := range opts {
		opt(&mf)
	}
	if max := len(data) * 64; max < size {
		size = max
	}
	var markLen int
	data, markLen = mf.f.encode(data, size)
	reader := bitstream.NewBitReader(data, 0, 0)
	reader.SetBits(markLen)
	return &Mark64{
		size:   size,
		reader: reader,
		mf:     mf,
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
	return m.size
}

func (m *Mark64) NewDecoder(bits []bool) watermark.MarkDecoder {
	w := bitstream.NewBitWriter[uint64](0, 0)
	for _, v := range bits {
		w.WriteBool(v)
	}
	reader := bitstream.NewBitReader(w.Data(), 0, 0)
	reader.SetBits(m.mf.f.encodedLen(m.size))
	return &Mark64{
		size:   m.size,
		mf:     m.mf,
		reader: reader,
	}
}

func (m *Mark64) DecodeToBytes() []byte {
	var decoded []byte
	_ = golay.DecodeBinay(m.reader.Data(), &decoded)
	return decoded[:m.size/8]
}

func (m *Mark64) DecodeToString() string {
	var decoded []byte
	_ = golay.DecodeBinay(m.reader.Data(), &decoded)
	return string(decoded[:m.size/8])
}

func (m *Mark64) DecodeToBools() []bool {
	var decoded []uint64
	_ = golay.DecodeBinay(m.reader.Data(), &decoded)
	reader := bitstream.NewBitReader(decoded, 0, 0)
	var data = make([]bool, m.size)
	for i := range data {
		data[i], _ = reader.ReadBitAt(i)
	}
	return data
}
