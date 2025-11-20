package mark

import (
	"math/rand"

	"github.com/yyyoichi/bitstream-go"
	"github.com/yyyoichi/golay"
)

var _ factroy = (*shuffledgolay)(nil)

type shuffledgolay int64

func (sg shuffledgolay) encode(data []uint64, size int) ([]uint64, int) {
	var encoded []uint64
	enc := golay.NewEncoder(&encoded)
	_ = enc.Encode(data, size)
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

func (sg shuffledgolay) decode(data []bool, size int) *bitstream.BitReader[uint64] {
	// reverse
	index := make([]int, len(data))
	for i := range index {
		index[i] = i
	}
	seed := int64(sg)
	rd := rand.New(rand.NewSource(seed))
	rd.Shuffle(len(index), func(i, j int) {
		index[i], index[j] = index[j], index[i]
	})

	w := bitstream.NewBitWriter[uint64](0, 0)
	for i, x := range index {
		w.WriteBitAt(x, data[i])
	}

	// decode
	var decoded []uint64
	dec := golay.NewDecoder(w.Data(), w.Bits())
	_ = dec.Decode(&decoded)

	r := bitstream.NewBitReader(decoded, 0, 0)
	r.SetBits(size)
	return r
}

func (sg shuffledgolay) encodedLen(size int) int {
	return golay.EncodedBits(size)
}

var _ factroy = (*withoutecc)(nil)

type withoutecc struct{}

func (we withoutecc) encode(data []uint64, size int) ([]uint64, int) {
	return data, size
}
func (we withoutecc) decode(data []bool, size int) *bitstream.BitReader[uint64] {
	w := bitstream.NewBitWriter[uint64](0, 0)
	for _, v := range data {
		w.WriteBool(v)
	}
	reader := bitstream.NewBitReader(w.Data(), 0, 0)
	reader.SetBits(size)
	return reader
}

func (we withoutecc) encodedLen(size int) int {
	return size
}
