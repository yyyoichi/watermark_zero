package mark

import (
	"testing"
)

func TestShuffledGolay(t *testing.T) {
	var sg shuffledgolay = 12345
	t.Run("encode length", func(t *testing.T) {
		for v := range 64 * 4 {
			_, l := sg.encode([]uint64{1, 2, 3, 4}, v)
			if l != sg.encodedLen(v) {
				t.Errorf("expected %d, got %d", sg.encodedLen(v), l)
			}
		}
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected panic for size exceeding data length")
			}
		}()
		sg.encode([]uint64{1, 2, 3, 4}, 64*4+1)
	})

	t.Run("encode/decode", func(t *testing.T) {
		original := []uint64{0x1234567890abcdef, 0xfedcba0987654321}
		size := 128
		encoded, _ := sg.encode(original, size)

		// Convert encoded data to bool slice
		reader := sg.decode(encoded, size)
		if reader.Bits() != size {
			t.Errorf("expected decoded bits %d, got %d", size, reader.Bits())
		}
		if reader.Read64R(64, 0) != original[0] {
			t.Errorf("expected first uint64 %x, got %x", original[0], reader.Read64R(64, 0))
		}
		if reader.Read64R(64, 1) != original[1] {
			t.Errorf("expected second uint64 %x, got %x", original[1], reader.Read64R(64, 1))
		}
	})
}
