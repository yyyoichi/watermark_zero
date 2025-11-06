package watermark

import (
	"image"
	"image/color"

	"github.com/yyyoichi/watermark_zero/internal/kmeans"
	"github.com/yyyoichi/watermark_zero/internal/yuv"
)

type imageCore struct {
	bounds                image.Rectangle
	width, height         int
	area                  int
	waveWidth, waveHeight int

	alpha []uint16
	// Y[]float32, U[]float32, V[]float32
	colors [][]float32
}

func newImageCore(src image.Image) imageCore {
	var c imageCore
	c.bounds = src.Bounds()
	c.width, c.height = c.bounds.Dx(), c.bounds.Dy()
	c.area = c.width * c.height
	c.colors = [][]float32{
		make([]float32, c.area), // Y
		make([]float32, c.area), // U
		make([]float32, c.area), // V
	}

	pixels := make([]color.Color, c.area)
	idx := 0
	for y := range c.height {
		for x := range c.width {
			pixels[idx] = src.At(x, y)
			idx++
		}
	}
	yuv.ColorToYUVBatch(pixels, c.colors[0], c.colors[1], c.colors[2], c.alpha)
	return c
}

func (c imageCore) copy() imageCore {
	tmp := [][]float32{
		make([]float32, c.area),
		make([]float32, c.area),
		make([]float32, c.area),
	}
	for i := range c.colors {
		_ = copy(tmp[i], c.colors[i])
	}
	c.colors = tmp
	return c
}

func (c imageCore) build() image.Image {
	var dist = image.NewRGBA64(c.bounds)
	pixels := make([]color.RGBA64, c.area)
	idx := 0
	yuv.YUVToRGBA64Batch(c.colors[0], c.colors[1], c.colors[2], c.alpha, pixels)
	for y := range c.height {
		for x := range c.width {
			dist.SetRGBA64(x, y, pixels[idx])
			idx++
		}
	}
	return dist
}

type blockShape [2]int

func newBlockShape(width, height int) blockShape {
	if width%2 != 0 {
		width += 1
	}
	if height%2 != 0 {
		height += 1
	}
	if width < 4 {
		width = 4
	}
	if height < 4 {
		height = 4
	}
	return [2]int{width / 2, height / 2}
}

func (s blockShape) blockArea() int {
	return s[0] * s[1]
}

func (s blockShape) totalBlocks(waveWidth, waveHeight int) int {
	return (waveWidth / s[0]) * (waveHeight / s[1])
}

type embedMark []bool

func (m embedMark) len() int {
	return len(m)
}

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
func (m extractMark) len() int {
	return len(m)
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
