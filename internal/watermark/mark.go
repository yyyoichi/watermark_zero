package watermark

import "github.com/yyyoichi/watermark_zero/internal/kmeans"

type embedMark []bool

func (m embedMark) getBit(at int) float64 {
	if m[at%len(m)] {
		return 1
	}
	return 0
}

type extractMark []kmeans.AverageStore

func newExtractMark(markLen int) extractMark {
	return make([]kmeans.AverageStore, markLen)
}

func (m extractMark) setBit(at int, v float64) {
	m[at%len(m)].Add(v)
}

func (m extractMark) averages() []float64 {
	avrs := make([]float64, len(m))
	for i := range m {
		avrs[i] = m[i].Average()
	}
	return avrs
}
