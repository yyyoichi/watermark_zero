package shuffle

import (
	"math/rand"
	"testing"
)

func TestShuffle(t *testing.T) {
	for range 100 {
		l := rand.Intn(100_000) + 1
		data := make([]any, l)
		for i := range data {
			data[i] = i
		}
		Shuffle(data)
		Ishuffle(data)
		for i := range data {
			if data[i] != i {
				t.Fatalf("mismatch at index %d: got %v, want %d", i, data[i], i)
			}
		}
	}
}
