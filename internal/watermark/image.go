package watermark

import (
	"image"
	"image/color"

	"github.com/yyyoichi/watermark_zero/internal/yuv"
)

type ImageSource struct {
	bounds                image.Rectangle
	width, height         int
	area                  int
	waveWidth, waveHeight int

	alpha []uint16
	// Y[]float32, U[]float32, V[]float32
	colors [][]float32
}

func NewImageCore(src image.Image) ImageSource {
	var s ImageSource
	s.bounds = src.Bounds()
	s.width, s.height = s.bounds.Dx(), s.bounds.Dy()
	s.waveWidth, s.waveHeight = (s.width+1)/2, (s.height+1)/2
	s.area = s.width * s.height
	s.colors = [][]float32{
		make([]float32, s.area), // Y
		make([]float32, s.area), // U
		make([]float32, s.area), // V
	}
	s.alpha = make([]uint16, s.area)

	pixels := make([]color.Color, s.area)
	idx := 0
	for y := range s.height {
		for x := range s.width {
			pixels[idx] = src.At(x, y)
			idx++
		}
	}
	yuv.ColorToYUVBatch(pixels, s.colors[0], s.colors[1], s.colors[2], s.alpha)
	return s
}

func (s ImageSource) Copy() ImageSource {
	tmp := [][]float32{
		make([]float32, s.area),
		make([]float32, s.area),
		make([]float32, s.area),
	}
	for i := range s.colors {
		_ = copy(tmp[i], s.colors[i])
	}
	s.colors = tmp
	return s
}

func (s ImageSource) build() image.Image {
	var dist = image.NewRGBA64(s.bounds)
	pixels := make([]color.RGBA64, s.area)
	idx := 0
	yuv.YUVToRGBA64Batch(s.colors[0], s.colors[1], s.colors[2], s.alpha, pixels)
	for y := range s.height {
		for x := range s.width {
			dist.SetRGBA64(x, y, pixels[idx])
			idx++
		}
	}
	return dist
}
