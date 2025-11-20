package mark

import (
	"math/rand"

	"github.com/yyyoichi/bitstream-go"
	"github.com/yyyoichi/golay"
)

var _ factroy = (*shuffledgolay)(nil)

type shuffledgolay int64

func (sg shuffledgolay) encode(data []uint64, size int) ([]uint64, int) {
	if size == 0 {
		return nil, 0
	}
	if size > len(data)*64 {
		// implies that size does not exceed data length
		panic("size exceeds data length")
	}
	var encoded []uint64
	enc := golay.NewEncoder(&encoded)
	_ = enc.Encode(data, size)
	encodedLen := enc.Bits()
	// shuffle
	index := sg.generatePermutation(encodedLen)

	// Apply permutation
	r := bitstream.NewBitReader(encoded, 0, 0)
	w := bitstream.NewBitWriter[uint64](0, 0)
	for i := range encodedLen {
		bit, _ := r.ReadBitAt(index[i])
		w.WriteBitAt(i, bit)
	}
	return w.Data(), encodedLen
}

func (sg shuffledgolay) decode(data []bool, size int) *bitstream.BitReader[uint64] {
	// reverse shuffle: create same permutation then apply inverse
	index := sg.generatePermutation(len(data))

	// Apply inverse permutation
	w := bitstream.NewBitWriter[uint64](0, 0)
	for i := range data {
		w.WriteBitAt(index[i], data[i])
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

func (sg shuffledgolay) generatePermutation(length int) []int {
	index := make([]int, length)
	for i := range index {
		index[i] = i
	}
	seed := int64(sg)
	rd := rand.New(rand.NewSource(seed))
	rd.Shuffle(length, func(i, j int) {
		index[i], index[j] = index[j], index[i]
	})
	return index
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
