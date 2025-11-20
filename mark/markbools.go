package mark

import "github.com/yyyoichi/bitstream-go"

func NewBools(data []bool, opts ...Option) *Mark64 {
	w := bitstream.NewBitWriter[uint64](0, 0)
	for _, v := range data {
		w.WriteBool(v)
	}
	return New64(w.Data(), w.Bits(), opts...)
}
