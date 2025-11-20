package watermark

type EmbedMark interface {
	GetBit(at int) float64
	Len() int
	ExtractSize() int
}
