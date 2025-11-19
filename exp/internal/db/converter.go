package db

import (
	"bytes"
	"encoding/binary"
)

// BoolSliceToBytes converts []bool to []byte
func BoolSliceToBytes(bits []bool) []byte {
	numBytes := (len(bits) + 7) / 8
	data := make([]byte, numBytes)
	for i, bit := range bits {
		if bit {
			data[i/8] |= 1 << (7 - i%8)
		}
	}
	return data
}

// BytesToBoolSlice converts []byte to []bool
func BytesToBoolSlice(data []byte, bitLength int) []bool {
	bits := make([]bool, bitLength)
	for i := 0; i < bitLength && i < len(data)*8; i++ {
		bits[i] = (data[i/8] & (1 << (7 - i%8))) != 0
	}
	return bits
}

// Uint64SliceToBytes converts []uint64 to []byte
func Uint64SliceToBytes(data []uint64) []byte {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, data)
	return buf.Bytes()
}

// BytesToUint64Slice converts []byte to []uint64
func BytesToUint64Slice(data []byte) []uint64 {
	count := len(data) / 8
	result := make([]uint64, count)
	buf := bytes.NewReader(data)
	binary.Read(buf, binary.LittleEndian, &result)
	return result
}
