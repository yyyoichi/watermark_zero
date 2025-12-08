package shuffle

import (
	"math/rand"
	"slices"
)

func Shuffle[T any](data []T) {
	rd := rand.New(rand.NewSource(1234))
	rd.Shuffle(len(data), func(i, j int) {
		data[i], data[j] = data[j], data[i]
	})
}

func Ishuffle[T any](data []T) {
	index := make([]int, len(data))
	for i := range index {
		index[i] = i
	}
	rd := rand.New(rand.NewSource(1234))
	rd.Shuffle(len(index), func(i, j int) {
		index[i], index[j] = index[j], index[i]
	})

	cp := slices.Clone(data)
	for i, x := range index {
		data[x] = cp[i]
	}
}
