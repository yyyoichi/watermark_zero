package bitconv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBitConv(t *testing.T) {
	test := []struct {
		data []byte
		exp  []byte
	}{
		{data: []byte{0b10101010}, exp: []byte{0b10101010}},
		{data: []byte{0b11110000, 0b00001111}, exp: []byte{0b11110000, 0b00001111}},
		{data: []byte("Hello"), exp: []byte("Hello")},
		{data: []byte("こんにちは"), exp: []byte("こんにちは")},
		{data: []byte("🐣"), exp: []byte("🐣")},
		{data: []byte{}, exp: []byte{}},
	}
	for _, tt := range test {
		bits := BytesToBools(tt.data)
		out := BoolsToBytes(bits)
		assert.Equal(t, tt.exp, out)
	}
}
