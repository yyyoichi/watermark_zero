package mark

import (
	"github.com/yyyoichi/bitstream-go"
	watermark "github.com/yyyoichi/watermark_zero"
)

var _ watermark.MarkCore = (*Mark64)(nil)
var _ watermark.EmbedMark = (*Mark64)(nil)
var _ watermark.ExtractMark = (*Mark64)(nil)
var _ watermark.MarkDecoder = (*Mark64)(nil)

// Mark64 is a struct that implements EmbedMark, ExtractMark, and MarkDecoder interfaces.
// It manages embedded mark bits based on uint64.
type Mark64 struct {
	size   int
	reader *bitstream.BitReader[uint64]
	mf     markFactory
}

// new64 initializes and returns a new Mark64 instance.
// It receives mark data as []uint64, where size specifies the number of valid bits.
// By default, it applies Golay23 encoding to add redundancy to the mark.
// If data is empty, it uses "Hello World" represented as []uint64 as the mark.
func new64(data []uint64, size int, opts ...Option) *Mark64 {
	if len(opts) == 0 {
		opts = append(opts, WithGolay(DefaultShuffleSeed))
	}
	var mf markFactory
	for _, opt := range opts {
		opt(&mf)
	}
	if len(data) == 0 {
		return NewBytes([]byte("Hello World"), opts...)
	}
	if max := len(data) * 64; max < size || size < 1 {
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

// NewExtract receives the bit length of the embedded mark and returns an interface for extracting watermarks.
// Extraction requires the same size and opts as used during embedding.
func NewExtract(size int, opts ...Option) watermark.ExtractMark {
	if len(opts) == 0 {
		opts = append(opts, WithGolay(DefaultShuffleSeed))
	}
	var mf markFactory
	for _, opt := range opts {
		opt(&mf)
	}
	return &Mark64{
		size: size,
		mf:   mf,
	}
}

// GetBit returns the bit value at the specified position as a float64.
// The position wraps around using modulo if it exceeds the mark length.
func (m *Mark64) GetBit(at int) float64 {
	n := at % m.reader.Bits()
	return float64(m.reader.Read8R(1, n))
}

// Len returns the bit length of the encoded mark after applying error correction.
// This is typically used internally and rarely needs to be called directly by users.
func (m *Mark64) Len() int {
	return m.mf.f.encodedLen(m.size)
}

// ExtractSize returns the bit length required for watermark extraction.
// For bool marks, this is the slice length; for string marks, it's len([]byte(str)) * 8;
// for byte marks, it's len(bytes) * 8.
func (m *Mark64) ExtractSize() int {
	return m.size
}

// NewDecoder receives the extracted bit sequence from the watermark and initializes a MarkDecoder.
// This is typically used internally and rarely needs to be called directly by users.
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

// DecodeToBytes decodes the extracted watermark data and returns it as a byte slice.
func (m *Mark64) DecodeToBytes() []byte {
	r := m.mf.f.decode(m.reader.Data(), m.size)
	var decoded = make([]byte, (m.size+7)/8)
	for i := range decoded {
		decoded[i] = r.Read8R(8, i)
	}
	return decoded
}

// DecodeToString decodes the extracted watermark data and returns it as a string.
// It internally calls DecodeToBytes and converts the result to a string.
func (m *Mark64) DecodeToString() string {
	return string(m.DecodeToBytes())
}

// DecodeToBools decodes the extracted watermark data and returns it as a boolean slice.
// Each element represents a single bit of the original mark.
func (m *Mark64) DecodeToBools() []bool {
	r := m.mf.f.decode(m.reader.Data(), m.size)
	_ = r.Seek(0)
	var decoded = make([]bool, m.size)
	for i := range decoded {
		decoded[i], _ = r.ReadBit()
	}
	return decoded
}
