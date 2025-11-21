package mark

import (
	"exp/internal/shuffle"

	"github.com/yyyoichi/bitstream-go"
	"github.com/yyyoichi/golay"
)

type Mark struct {
	Name     string
	Original []bool
	Encoded  []bool

	Decode func([]bool) []bool
}

func NewNormalMark(original []bool) Mark {
	m := Mark{
		Name:     "Normal",
		Original: original,
		Encoded:  original,
		Decode: func(b []bool) []bool {
			return b
		},
	}
	return m
}

func NewShuffledGolayMark(original []bool) Mark {
	tmp := NewGolayMark(original)
	m := Mark{
		Name:     "SfGolay",
		Original: tmp.Original,
		Encoded:  tmp.Encoded,
	}
	shuffle.Shuffle(m.Encoded)
	m.Decode = func(b []bool) []bool {
		shuffle.Ishuffle(b)
		return tmp.Decode(b)
	}
	return m
}

func NewGolayMark(original []bool) Mark {
	l := len(original)

	var m Mark
	m.Name = "Golay"
	m.Original = original
	{
		w := bitstream.NewBitWriter[uint64](0, 0)
		for _, v := range original {
			w.WriteBool(v)
		}
		var encoded []uint64
		enc := golay.NewEncoder(&encoded)
		_ = enc.Encode(w.Data(), w.Bits())
		r := bitstream.NewBitReader(encoded, 0, 0)
		r.SetBits(enc.Bits())
		m.Encoded = make([]bool, enc.Bits())
		for i := range m.Encoded {
			m.Encoded[i] = r.Read8R(1, i) == 1
		}
	}
	m.Decode = func(b []bool) []bool {
		w := bitstream.NewBitWriter[uint64](0, 0)
		for _, v := range b {
			w.WriteBool(v)
		}
		data := w.Data()
		var decoded []uint64
		dec := golay.NewDecoder(data, len(b))
		_ = dec.Decode(&decoded)
		r := bitstream.NewBitReader(decoded, 0, 0)
		r.SetBits(dec.Bits())
		result := make([]bool, l)
		for i := range result {
			result[i] = r.Read8R(1, i) == 1
		}
		return result
	}
	return m
}
