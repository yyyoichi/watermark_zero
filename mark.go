package watermark

// MarkCore defines the core interface for mark operations.
// It provides methods to get the encoded mark length and the extraction size.
type MarkCore interface {
	// Len returns the bit length of the encoded mark after applying error correction.
	Len() int
	// ExtractSize returns the bit length required for watermark extraction.
	ExtractSize() int
}

// EmbedMark defines the interface for embedding watermarks.
// It provides methods to retrieve mark bits for embedding into media.
type EmbedMark interface {
	// GetBit returns the bit value at the specified position as a float64.
	GetBit(at int) float64
	MarkCore
}

// ExtractMark defines the interface for extracting watermarks.
// It provides methods to initialize a decoder from extracted bit sequences.
type ExtractMark interface {
	// NewDecoder receives the extracted bit sequence and initializes a MarkDecoder.
	// Each byte in the slice represents a single bit (0 or 1).
	NewDecoder([]byte) MarkDecoder
	MarkCore
}

// MarkDecoder defines the interface for decoding extracted watermark data.
// It provides methods to convert the decoded data into various formats.
type MarkDecoder interface {
	// DecodeToBytes decodes the watermark data and returns it as a byte slice.
	DecodeToBytes() []byte
	// DecodeToString decodes the watermark data and returns it as a string.
	DecodeToString() string
}
