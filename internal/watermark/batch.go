package watermark

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"sync"

	"github.com/yyyoichi/watermark_zero/internal/dct"
	"github.com/yyyoichi/watermark_zero/internal/dwt"
	"github.com/yyyoichi/watermark_zero/internal/kmeans"
	"github.com/yyyoichi/watermark_zero/internal/svd"
	"github.com/yyyoichi/watermark_zero/internal/yuv"
)

type ExtractBatch struct {
	// src wavelets
	waveWidth, waveHeight int
	// [yuv][rowmajar]
	getCA func() [3][]float32
	// s value getter by request shape
	data sync.Map
}

func NewExtractBatch(src image.Image) *ExtractBatch {
	var b ExtractBatch
	b.getCA = sync.OnceValue(func() [3][]float32 {
		return b.initCA(src)
	})
	bound := src.Bounds()
	w, h := bound.Dx(), bound.Dy()
	b.waveWidth, b.waveHeight = (w+1)/2, (h+1)/2
	return &b
}

func (b *ExtractBatch) TotalBlock(shape BlockShape) int {
	return shape.totalBlocks(b.waveWidth, b.waveHeight)
}

func (b *ExtractBatch) Extract(ctx context.Context, markLen int, shape BlockShape, d1, d2 int) ([]bool, error) {
	totalBlocks := shape.totalBlocks(b.waveWidth, b.waveHeight)
	mk := newExtractMark(markLen)
	var extract func(s0, s1 float64) (v float64)
	if d2 < 1 {
		extract = func(s0, s1 float64) (v float64) {
			if int(s0)%d1 > d1/2 {
				return 1
			}
			return 0
		}
	} else {
		extract = func(s0, s1 float64) (v float64) {
			if int(s0)%d1 > d1/2 {
				v = 1
			}
			if int(s1)%d2 > d2/2 {
				return (v*3 + 1) / 4.
			}
			return (v * 3) / 4.
		}
	}
	data := b.ensureSValue(shape)
	for _, blocks := range data {
		for at := range totalBlocks {
			if blocks[at][0] == -1 {
				continue
			}
			v := extract(blocks[at][0], blocks[at][1])
			mk.setBit(at, v)
		}
	}
	avrs := mk.averages()
	return kmeans.OneDimKmeans(avrs), nil
}

func (b *ExtractBatch) initCA(src image.Image) (output [3][]float32) {
	bound := src.Bounds()
	w, h := bound.Dx(), bound.Dy()
	ps := w * h
	colors := [][]float32{
		make([]float32, ps), // Y
		make([]float32, ps), // U
		make([]float32, ps), // V
	}
	pixels := make([]color.Color, ps)
	idx := 0
	for y := range h {
		for x := range w {
			pixels[idx] = src.At(x, y)
			idx++
		}
	}
	yuv.ColorToYUVBatch(pixels, colors[0], colors[1], colors[2], make([]uint16, ps))

	var wg sync.WaitGroup
	for level, c := range colors {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wavelets := dwt.HaarDWT(c, w, nil)
			output[level] = wavelets[0]
		}()
	}
	wg.Wait()
	return
}

// [yuv][block rowmajar][s valud 0,1]
// [2]float64{-1, 0} -> ignore
type sValueChunk = [3][][2]float64

func (b *ExtractBatch) ensureSValue(shape BlockShape) sValueChunk {
	key := fmt.Sprintf("%d-%d", shape.width(), shape.height())
	fn, _ := b.data.LoadOrStore(key, sync.OnceValue(func() sValueChunk {
		indexes := dwt.NewBlockMap(b.waveWidth, b.waveHeight, shape.width(), shape.height()).GetMap()
		totalBlocks := shape.totalBlocks(b.waveWidth, b.waveHeight)
		blockArea := shape.blockArea()
		dct := dct.New(shape.width(), shape.height())
		svd := svd.New(shape.width(), shape.height())
		cAs := b.getCA()
		byLevel := sValueChunk{
			make([][2]float64, totalBlocks),
			make([][2]float64, totalBlocks),
			make([][2]float64, totalBlocks),
		}
		for level, cA := range cAs {
			// block row majar cA
			sorted := make([]float32, len(cA))
			for i, v := range cA {
				sorted[indexes[i]] = v
			}
			for at := range totalBlocks {
				data := sorted[at*blockArea : (at+1)*blockArea : (at+1)*blockArea]
				d, _ := dct.Exec(data)
				s, _, err := svd.Exec(d)
				if err != nil {
					byLevel[level][at][0] = -1
					continue
				}
				byLevel[level][at][0] = s[0]
				byLevel[level][at][1] = s[1]
			}
		}
		return byLevel
	}))
	return fn.(func() sValueChunk)()
}
