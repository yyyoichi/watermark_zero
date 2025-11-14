package bench_test

import (
	"image"
	"image/color"
	"testing"

	watermark "github.com/yyyoichi/watermark_zero"
)

// BenchmarkEmbed_FHD_Table runs a table-driven set of embed benchmarks for FHD images
func BenchmarkEmbed_FHD(b *testing.B) {
	test := []struct {
		name string
		opts []watermark.Option
	}{
		{name: "4x4_D1", opts: []watermark.Option{
			watermark.WithBlockShape(4, 4),
			watermark.WithD1(36),
		}},
		{name: "4x4_D1D2", opts: []watermark.Option{
			watermark.WithBlockShape(4, 4),
			watermark.WithD1D2(36, 20),
		}},
		{name: "8x8_D1", opts: []watermark.Option{
			watermark.WithBlockShape(8, 8),
			watermark.WithD1(36),
		}},
		{name: "8x8_D1D2", opts: []watermark.Option{
			watermark.WithBlockShape(8, 8),
			watermark.WithD1D2(36, 20),
		}},
		{name: "12x12_D1", opts: []watermark.Option{
			watermark.WithBlockShape(12, 12),
			watermark.WithD1(36),
		}},
		{name: "12x12_D1D2", opts: []watermark.Option{
			watermark.WithBlockShape(12, 12),
			watermark.WithD1D2(36, 20),
		}},
		{name: "16x16_D1", opts: []watermark.Option{
			watermark.WithBlockShape(16, 16),
			watermark.WithD1(36),
		}},
		{name: "16x16_D1D2", opts: []watermark.Option{
			watermark.WithBlockShape(16, 16),
			watermark.WithD1D2(36, 20),
		}},
	}

	img := createImage(1920, 1080)
	mark := createTestMark()
	ctx := b.Context()

	for _, tt := range test {
		b.Run(tt.name, func(b *testing.B) {
			// Initialize watermark instance for this case
			w, err := watermark.New(tt.opts...)
			if err != nil {
				b.Fatalf("Failed to create Watermark instance (%s): %v", tt.name, err)
			}
			for b.Loop() {
				dist, err := w.Embed(ctx, img, mark)
				if err != nil {
					b.Fatalf("Failed to embed watermark (%s): %v", tt.name, err)
				}
				_ = dist
			}
		})
	}
}

// createImage creates a widthxheight test image with gradient pattern
func createImage(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			// Create gradient effect to simulate realistic image data
			r := uint8((x * 255) / width)
			g := uint8((y * 255) / height)
			b := uint8(((x + y) * 255) / (width + height))
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}
	return img
}

// createTestMark creates a test watermark bit sequence
func createTestMark() watermark.EmbedMark {
	return watermark.NewBoolEmbedMark([]bool{
		true, false, true, true, false, false, true, false,
		false, true, false, true, true, false, true, true,
		true, true, false, false, true, false, false, true,
		false, false, true, true, false, true, true, false,
	})
}
