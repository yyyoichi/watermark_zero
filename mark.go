package watermark

type MarkCore interface {
	Len() int
	ExtractSize() int
}

type EmbedMark interface {
	GetBit(at int) float64
	MarkCore
}

type ExtractMark interface {
	NewDecoder([]bool) MarkDecoder
	MarkCore
}

type MarkDecoder interface {
	DecodeToBytes() []byte
	DecodeToString() string
	DecodeToBools() []bool
}
