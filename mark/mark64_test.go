package mark

import (
	"testing"

	"github.com/stretchr/testify/assert"
	watermark "github.com/yyyoichi/watermark_zero"
)

func TestMark64EncodeDecode(t *testing.T) {
	test := []struct {
		name   string
		new    func(...Option) *Mark64
		assert func(*testing.T, watermark.MarkDecoder)
	}{
		{"string_1",
			func(o ...Option) *Mark64 {
				return NewString("TEST_MARK", o...)
			},
			func(t *testing.T, m watermark.MarkDecoder) {
				assert.Equal(t, "TEST_MARK", m.DecodeToString())
				assert.Equal(t, []byte("TEST_MARK"), m.DecodeToBytes())
			}},
		{"string_2",
			func(o ...Option) *Mark64 {
				return NewString("a", o...)
			},
			func(t *testing.T, m watermark.MarkDecoder) {
				assert.Equal(t, "a", m.DecodeToString())
				assert.Equal(t, []byte("a"), m.DecodeToBytes())
			}},
		{"string_3",
			func(o ...Option) *Mark64 {
				return NewString("こんにちはHello", o...)
			},
			func(t *testing.T, m watermark.MarkDecoder) {
				assert.Equal(t, "こんにちはHello", m.DecodeToString())
				assert.Equal(t, []byte("こんにちはHello"), m.DecodeToBytes())
			}},
		{"bytes_1",
			func(o ...Option) *Mark64 {
				return NewBytes([]byte{0x01, 0xff, 0x00}, o...)
			},
			func(t *testing.T, m watermark.MarkDecoder) {
				assert.Equal(t, []byte{0x01, 0xff, 0x00}, m.DecodeToBytes())
			},
		},
		{"bytes_2",
			func(o ...Option) *Mark64 {
				return NewBytes([]byte("hello world!"), o...)
			},
			func(t *testing.T, m watermark.MarkDecoder) {
				assert.Equal(t, []byte("hello world!"), m.DecodeToBytes())
			},
		},
		{"bools_1",
			func(o ...Option) *Mark64 {
				return NewBools([]bool{true}, o...)
			},
			func(t *testing.T, m watermark.MarkDecoder) {
				assert.Equal(t, []byte{0b10_000_000}, m.DecodeToBytes())
			},
		},
		{"bools_2",
			func(o ...Option) *Mark64 {
				return NewBools([]bool{false}, o...)
			},
			func(t *testing.T, m watermark.MarkDecoder) {
				assert.Equal(t, []byte{0}, m.DecodeToBytes())
			},
		},
		{"bools_3",
			func(o ...Option) *Mark64 {
				return NewBools([]bool{
					false, true, false, true,
					false, false, true, true,
					false, false, false, true, true, true,
				}, o...)
			},
			func(t *testing.T, m watermark.MarkDecoder) {
				assert.Equal(t, []byte{0b01_010_011, 0b00_011_100}, m.DecodeToBytes())
			},
		},
		{"empty_string",
			func(o ...Option) *Mark64 {
				return NewBools([]bool{}, o...)
			},
			func(t *testing.T, m watermark.MarkDecoder) {
				assert.Equal(t, "Hello World", m.DecodeToString())
			},
		},
		{"empty_bytes",
			func(o ...Option) *Mark64 {
				return NewBools([]bool{}, o...)
			},
			func(t *testing.T, m watermark.MarkDecoder) {
				assert.Equal(t, "Hello World", m.DecodeToString())
			},
		},
		{"empty_bools",
			func(o ...Option) *Mark64 {
				return NewBools([]bool{}, o...)
			},
			func(t *testing.T, m watermark.MarkDecoder) {
				assert.Equal(t, "Hello World", m.DecodeToString())
			},
		},
	}
	noPanicDecodes := func(t *testing.T, mark watermark.MarkDecoder) {
		assert.NotPanics(t, func() { mark.DecodeToString() })
		assert.NotPanics(t, func() { mark.DecodeToBytes() })
	}
	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			for _, opt := range []Option{
				WithoutECC(),
				WithGolay(DefaultShuffleSeed),
			} {
				mark := tt.new(opt)
				assert.NotZero(t, mark.Len())
				assert.NotZero(t, mark.ExtractSize())
				noPanicDecodes(t, mark)
				assert.NotPanics(t, func() { mark.NewDecoder(nil) })
				tt.assert(t, mark)

				var data = make([]byte, mark.Len())
				for i := range data {
					if mark.GetBit(i) > 0 {
						data[i] = 1
					}
				}
				dec := mark.NewDecoder(data)
				noPanicDecodes(t, dec)
				tt.assert(t, dec)

				extr := NewExtract(mark.ExtractSize(), opt)
				assert.Equal(t, mark.ExtractSize(), extr.ExtractSize())
				assert.Equal(t, mark.Len(), extr.Len())

				dec = extr.NewDecoder(data)
				noPanicDecodes(t, dec)
				tt.assert(t, dec)
			}
		})
	}
}
