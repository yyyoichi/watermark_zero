package strmark

import "github.com/yyyoichi/watermark_zero/internal/bitconv"

// Encode encodes the input string into a slice of booleans representing bits.
func Encode(src string) []bool {
	return bitconv.BytesToBools([]byte(src))
}

// Decode decodes the input slice of booleans back into the original string.
func Decode(mark []bool) string {
	return string(bitconv.BoolsToBytes(mark))
}

var _ Mark = (*StrMark)(nil)

type StrMark struct {
}

func New() Mark {
	return &StrMark{}
}

func (sm *StrMark) Encode(src string) ([]bool, error) {
	return bitconv.BytesToBools([]byte(src)), nil
}

func (sm *StrMark) Decode(mark []bool) (src string, err error) {
	return string(bitconv.BoolsToBytes(mark)), nil
}
