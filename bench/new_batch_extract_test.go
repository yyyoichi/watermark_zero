package bench_test

import (
	"sync"
	"testing"

	watermark "github.com/yyyoichi/watermark_zero"
)

func BenchmarkExtractBatch(b *testing.B) {
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
	markLen := 83 * 8
	ctx := b.Context()
	b.Run("v1", func(b *testing.B) {
		for b.Loop() {
			batch := watermark.NewBatch(img)
			var wg sync.WaitGroup
			wg.Add(len(test))
			for _, tt := range test {
				go func() {
					defer wg.Done()
					result, _ := batch.Extract(ctx, markLen, tt.opts...)
					_ = result
				}()
			}
			wg.Wait()
		}
	})
	b.Run("v2", func(b *testing.B) {
		for b.Loop() {
			batch := watermark.NewExtractBatch(img)
			var wg sync.WaitGroup
			wg.Add(len(test))
			for _, tt := range test {
				go func() {
					defer wg.Done()
					result, _ := batch.Extract(ctx, markLen, tt.opts...)
					_ = result
				}()
			}
			wg.Wait()
		}
	})
}
