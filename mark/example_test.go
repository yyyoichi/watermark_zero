package mark_test

import (
	"fmt"

	"github.com/yyyoichi/watermark_zero/mark"
)

// ExampleNewString demonstrates how to create a mark from a string and decode it back.
func ExampleNewString() {
	// Create a mark from a string
	mark := mark.NewString("Hello")

	// ExtractSize equals len([]byte("Hello")) * 8
	fmt.Printf("Extract size: %d bits (= %d bytes * 8)\n", mark.ExtractSize(), len([]byte("Hello")))

	// Decode back to string
	decoded := mark.DecodeToString()
	fmt.Println(decoded)
	// Output:
	// Extract size: 40 bits (= 5 bytes * 8)
	// Hello
}

// ExampleNewBytes demonstrates how to create a mark from bytes and decode it back.
func ExampleNewBytes() {
	// Create a mark from bytes
	data := []byte{0x48, 0x69} // "Hi"
	mark := mark.NewBytes(data)

	// ExtractSize equals len(data) * 8
	fmt.Printf("Extract size: %d bits (= %d bytes * 8)\n", mark.ExtractSize(), len(data))

	// Decode back to bytes
	decoded := mark.DecodeToBytes()
	fmt.Printf("%s\n", decoded)
	// Output:
	// Extract size: 16 bits (= 2 bytes * 8)
	// Hi
}

// ExampleNewBools demonstrates how to create a mark from boolean slice.
func ExampleNewBools() {
	// Create a mark from boolean slice
	bools := []bool{true, false, true, true}
	mark := mark.NewBools(bools)

	// ExtractSize equals len(bools)
	fmt.Printf("Extract size: %d bits (= %d bools)\n", mark.ExtractSize(), len(bools))

	// Decode back to bytes (bools are packed into bytes)
	decoded := mark.DecodeToBytes()
	fmt.Printf("%08b\n", decoded[0])
	// Output:
	// Extract size: 4 bits (= 4 bools)
	// 10110000
}

// ExampleNewExtract demonstrates how to extract and decode a watermark.
func ExampleNewExtract() {
	// First, create and embed a mark
	embedMark := mark.NewString("Test")
	size := embedMark.ExtractSize()

	// Simulate extracting bits (in real scenario, these come from the watermarked image)
	// Each byte represents a single bit (0 or 1)
	extractedBits := make([]byte, embedMark.Len())
	for i := range extractedBits {
		if embedMark.GetBit(i) > 0 {
			extractedBits[i] = 1
		}
	}

	// Create an extract interface for decoding
	extractMark := mark.NewExtract(size)

	// Decode the extracted bits
	decoder := extractMark.NewDecoder(extractedBits)
	decoded := decoder.DecodeToString()
	fmt.Println(decoded)
	// Output:
	// Test
}
