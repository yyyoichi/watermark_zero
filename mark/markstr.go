package mark

import "github.com/yyyoichi/bitstream-go"

func NewStr(data string, opts ...Option) *Mark64 {
	w := bitstream.NewBitWriter[uint64](0, 0)
	for _, v := range []byte(data) {
		w.Write8(0, 8, v)
	}
	return New64(w.Data(), w.Bits(), opts...)
}
