package watermark

import (
	"context"
	"fmt"
	"image"
	"sync"

	"github.com/yyyoichi/watermark_zero/internal/dct"
	"github.com/yyyoichi/watermark_zero/internal/dwt"
	"github.com/yyyoichi/watermark_zero/internal/kmeans"
	"github.com/yyyoichi/watermark_zero/internal/svd"
)

func Enable(src ImageSource, markLen int, shape BlockShape) error {
	if total := shape.TotalBlocks(src); total < markLen {
		return fmt.Errorf("total blocks %d < mark length %d", total, markLen)
	}
	return nil
}

func TotalBlocks(rect image.Rectangle, shape BlockShape) int {
	return shape.totalBlocks((rect.Dx()+1)/2, (rect.Dy()+1)/2)
}

func Wavelets(src ImageSource) []*dwt.Wavelets {
	var wavelets = make([]*dwt.Wavelets, 3)
	var wg sync.WaitGroup
	wg.Add(3)
	for yuv := range 3 {
		go func(yuv int) {
			defer wg.Done()
			wavelets[yuv] = dwt.New(src.colors[yuv], src.width)
		}(yuv)
	}
	wg.Wait()
	return wavelets
}

func Embed(ctx context.Context, src ImageSource, mark []bool, shape BlockShape, d1 int, d2 int, wavelets []*dwt.Wavelets, dctCache *dct.Cache) (image.Image, error) {
	var (
		totalBlocks = shape.TotalBlocks(src)
		blockArea   = shape.blockArea()
		mk          = embedMark(mark)
	)

	var embed func(s0, s1, bit float64) (r0 float64, r1 float64)
	if d2 < 1 {
		fd1 := float64(d1)
		embed = func(s0, s1, bit float64) (r0 float64, r1 float64) {
			r0 = (float64(int(s0)/d1) + 1.0/4.0 + 1.0/2.0*0.5*bit) * fd1
			r1 = s1
			return
		}
	} else {
		fd1, fd2 := float64(d1), float64(d2)
		embed = func(s0, s1, bit float64) (r0 float64, r1 float64) {
			r0 = (float64(int(s0)/d1) + 1.0/4.0 + 1.0/2.0*0.5*bit) * fd1
			r1 = (float64(int(s1)/d2) + 1.0/4.0 + 1.0/2.0*0.5*bit) * fd2
			return
		}
	}

	var (
		indexMap = dwt.NewBlockMap(src.waveWidth, src.waveHeight, shape.width(), shape.height()).GetMap()
		svd      = svd.New(shape.width(), shape.height())
	)
	var wave = func(yuv int) [][]float32 {
		return wavelets[yuv].Get(indexMap)
	}
	if wavelets == nil || len(wavelets) != 3 {
		wave = func(yuv int) [][]float32 {
			return dwt.HaarDWT(src.colors[yuv], src.width, indexMap)
		}
	}
	var dcos *dct.DCT
	if dctCache == nil {
		dcos = dct.New(shape.width(), shape.height())
	} else {
		dcos = dctCache.New(shape.width(), shape.height())
	}

	var wg sync.WaitGroup
	wg.Add(3)
	for yuv := range 3 {
		go func(yuv int) {
			defer wg.Done()
			// The wavelet transform rearranges the row-major slice into blocks that are also arranged in row-major order.
			// This is designed for efficient slice referencing without slice manipulation during transform and inverse transform operations.
			wavelets := wave(yuv)
			cA := wavelets[0]
			for at := range totalBlocks {
				data := cA[at*blockArea : (at+1)*blockArea : (at+1)*blockArea]
				bit := mk.getBit(at)
				d, idct := dcos.Exec(data)
				s, isvd, err := svd.Exec(d)
				if err != nil {
					return
				}
				s[0], s[1] = embed(s[0], s[1], bit)
				isvd()
				idct()
			}
			src.colors[yuv] = dwt.HaarIDWT(wavelets, src.width, src.height, indexMap)
		}(yuv)
	}
	wg.Wait()
	return src.build(), nil
}

func Extract(ctx context.Context, src ImageSource, markLen int, shape BlockShape, d1 int, d2 int, wavelets []*dwt.Wavelets, dctCache *dct.Cache) ([]bool, error) {
	var (
		totalBlocks = shape.TotalBlocks(src)
		blockArea   = shape.blockArea()
		mk          = newExtractMark(markLen)
	)

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

	var (
		indexMap = dwt.NewBlockMap(src.waveWidth, src.waveHeight, shape.width(), shape.height()).GetMap()
		svd      = svd.New(shape.width(), shape.height())
	)
	var wave = func(yuv int) [][]float32 {
		return wavelets[yuv].Get(indexMap)
	}
	if wavelets == nil || len(wavelets) != 3 {
		wave = func(yuv int) [][]float32 {
			return dwt.HaarDWT(src.colors[yuv], src.width, indexMap)
		}
	}
	var dcos *dct.DCT
	if dctCache == nil {
		dcos = dct.New(shape.width(), shape.height())
	} else {
		dcos = dctCache.New(shape.width(), shape.height())
	}

	var wg sync.WaitGroup
	wg.Add(3)
	for yuv := range 3 {
		go func(yuv int) {
			defer wg.Done()
			wavelets := wave(yuv)
			cA := wavelets[0]
			for at := range totalBlocks {
				data := cA[at*blockArea : (at+1)*blockArea : (at+1)*blockArea]
				d, _ := dcos.Exec(data)
				s, _, err := svd.Exec(d)
				if err != nil {
					return
				}
				v := extract(s[0], s[1])
				mk.setBit(at, v)
			}
			src.colors[yuv] = dwt.HaarIDWT(wavelets, src.width, src.height, indexMap)
		}(yuv)
	}
	wg.Wait()
	avrs := mk.averages()
	return kmeans.OneDimKmeans(avrs), nil
}
