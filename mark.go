package watermark

import "github.com/yyyoichi/watermark_zero/internal/watermark"

type EmbedMark = watermark.EmbedMark
type BitMark = watermark.BitMark

var _ EmbedMark = (*BoolEmbedMark)(nil)
var _ BitMark = (*BoolEmbedMark)(nil)

type BoolEmbedMark struct {
	data []bool
}

func NewBoolEmbedMark(data []bool) EmbedMark {
	return &BoolEmbedMark{data: data}
}

func (m *BoolEmbedMark) GetBit(at int) float64 {
	if m.data[at%len(m.data)] {
		return 1
	}
	return 0
}

func (m *BoolEmbedMark) Len() int {
	return len(m.data)
}
