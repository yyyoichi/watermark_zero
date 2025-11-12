package mark

import (
	"github.com/yyyoichi/bitstream-go"
	"github.com/yyyoichi/golay"
)

type Mark struct {
	Name     string
	Original []bool
	Encoded  []bool

	Decode func([]bool) []bool
}

func NewGolayMark(original []bool) Mark {
	l := len(original)

	var m Mark
	m.Name = "Golay"
	m.Original = original
	{
		w := bitstream.NewBitWriter[uint64](0, 0)
		for _, v := range original {
			w.Bool(v)
		}
		data, _ := w.Data()
		var encoded []uint64
		enc := golay.NewEncoder(data, l)
		_ = enc.Encode(&encoded)
		r := bitstream.NewBitReader(encoded, 0, 0)
		r.SetBits(enc.Bits())
		m.Encoded = make([]bool, enc.Bits())
		for i := range m.Encoded {
			m.Encoded[i] = r.U8R(1, i) == 1
		}
	}
	m.Decode = func(b []bool) []bool {
		w := bitstream.NewBitWriter[uint64](0, 0)
		for _, v := range b {
			w.Bool(v)
		}
		data, _ := w.Data()
		var decoded []uint64
		dec := golay.NewDecoder(data, len(b))
		_ = dec.Decode(&decoded)
		r := bitstream.NewBitReader(decoded, 0, 0)
		r.SetBits(dec.Bits())
		result := make([]bool, l)
		for i := range result {
			result[i] = r.U8R(1, i) == 1
		}
		return result
	}
	return m
}
