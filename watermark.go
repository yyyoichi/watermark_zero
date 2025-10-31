package watermark

import (
	"context"
	"errors"
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

var (
	ErrTooSmallImage = errors.New("image is too small for embedding or extracting")
)

type Watermark struct {
	blockShape [2]int // blockWidth, blockHeight
	embed      func(s0, s1, bit float64) (r0 float64, r1 float64)
	extract    func(s0, s1 float64) (v float64)
	dct        *dct.DCT
	svd        *svd.SVD
}

// New initializes a watermark processing structure.
// The blockShape and watermark coefficients d1, d2 can be optionally specified.
// For default values, refer to the init function.
func New(opts ...Option) (*Watermark, error) {
	w := new(Watermark)
	for _, opt := range opts {
		if err := opt(w); err != nil {
			return nil, err
		}
	}
	if err := w.init(); err != nil {
		return nil, err
	}
	return w, nil
}

// Embed embeds a bit sequence into an image.
//
// Process:
//  1. Converts the image to YUV color channels.
//  2. Applies Haar wavelet transform to each channel.
//  3. Divides the low-frequency region (cA) of each channel into blocks.
//  4. Embeds one bit per block using Discrete Cosine Transform and SVD.
//  5. Applies inverse transforms.
//  6. Reconstructs the image.
//
// Returns an error if the image is too small for the bit sequence to be embedded.
func (w *Watermark) Embed(ctx context.Context, src image.Image, mark []bool) (image.Image, error) {
	var (
		bounds                = src.Bounds()
		width, height         = bounds.Dx(), bounds.Dy()
		area                  = width * height
		waveWidth, waveHeight = (width + 1) / 2, (height + 1) / 2
		blockArea             = w.blockShape[0] * w.blockShape[1]
		totalBlocks           = (waveWidth / w.blockShape[0]) * (waveHeight / w.blockShape[1])
		markLen               = len(mark)
		getBit                = func(at int) float64 {
			if mark[at%markLen] {
				return 1
			}
			return 0
		}
	)
	if totalBlocks < markLen {
		return nil, fmt.Errorf("%w: total blocks %d < mark length %d", ErrTooSmallImage, totalBlocks, markLen)
	}

	var (
		alpha  = make([]uint16, area)
		colors = [][]float32{
			make([]float32, area), // Y
			make([]float32, area), // U
			make([]float32, area), // V
		}
	)
	{
		pixels := make([]color.Color, area)
		idx := 0
		for y := range height {
			for x := range width {
				pixels[idx] = src.At(x, y)
				idx++
			}
		}
		yuv.ColorToYUVBatch(pixels, colors[0], colors[1], colors[2], alpha)
	}
	var blockMap = dwt.NewBlockMap(waveWidth, waveHeight, w.blockShape[0], w.blockShape[1]).GetMap()
	var wg sync.WaitGroup
	wg.Add(3)
	for yuv := range 3 {
		go func(yuv int) {
			defer wg.Done()
			// The wavelet transform rearranges the row-major slice into blocks that are also arranged in row-major order.
			// This is designed for efficient slice referencing without slice manipulation during transform and inverse transform operations.
			wavelets := dwt.HaarDWT(colors[yuv], width, blockMap)
			colors[yuv] = nil
			cA := wavelets[0]
			for at := range totalBlocks {
				data := cA[at*blockArea : (at+1)*blockArea : (at+1)*blockArea]
				bit := getBit(at)
				d, idct := w.dct.Exec(data)
				s, isvd, err := w.svd.Exec(d)
				if err != nil {
					return
				}
				s[0], s[1] = w.embed(s[0], s[1], bit)
				isvd()
				idct()
			}
			colors[yuv] = dwt.HaarIDWT(wavelets, width, height, blockMap)
		}(yuv)
	}
	wg.Wait()

	var dist = image.NewRGBA64(bounds)
	{
		pixels := make([]color.RGBA64, area)
		idx := 0
		yuv.YUVToRGBA64Batch(colors[0], colors[1], colors[2], alpha, pixels)
		for y := range height {
			for x := range width {
				dist.SetRGBA64(x, y, pixels[idx])
				idx++
			}
		}
	}
	return dist, nil
}

// Extract extracts a bit sequence from an image.
//
// Process:
//  1. Converts the image to YUV color channels.
//  2. Applies Haar wavelet transform to each channel.
//  3. Divides the low-frequency region (cA) of each channel into blocks.
//  4. Extracts one bit per block using Discrete Cosine Transform and SVD.
//  5. Determines boolean values using k-means clustering on the average values of each block's bits.
//
// Returns an error if the image is too small for the expected bit sequence length.
func (w *Watermark) Extract(ctx context.Context, src image.Image, markLen int) ([]bool, error) {
	var (
		bounds                = src.Bounds()
		width, height         = bounds.Dx(), bounds.Dy()
		area                  = width * height
		waveWidth, waveHeight = (width + 1) / 2, (height + 1) / 2
		blockArea             = w.blockShape[0] * w.blockShape[1]
		totalBlocks           = (waveWidth / w.blockShape[0]) * (waveHeight / w.blockShape[1])
		dist                  = make([]kmeans.AverageStore, markLen)
		setValue              = func(at int, value float64) {
			dist[at%markLen].Add(value)
		}
	)
	if totalBlocks < markLen {
		return nil, fmt.Errorf("%w: total blocks %d < mark length %d", ErrTooSmallImage, totalBlocks, markLen)
	}
	var (
		alpha  = make([]uint16, area)
		colors = [][]float32{
			make([]float32, area), // Y
			make([]float32, area), // U
			make([]float32, area), // V
		}
	)
	{
		pixels := make([]color.Color, area)
		idx := 0
		for y := range height {
			for x := range width {
				pixels[idx] = src.At(x, y)
				idx++
			}
		}
		yuv.ColorToYUVBatch(pixels, colors[0], colors[1], colors[2], alpha)
	}
	var blockMap = dwt.NewBlockMap(waveWidth, waveHeight, w.blockShape[0], w.blockShape[1]).GetMap()
	var wg sync.WaitGroup
	wg.Add(3)
	for yuv := range 3 {
		go func(yuv int) {
			defer wg.Done()
			wavelets := dwt.HaarDWT(colors[yuv], width, blockMap)
			colors[yuv] = nil
			cA := wavelets[0]
			for at := range totalBlocks {
				data := cA[at*blockArea : (at+1)*blockArea : (at+1)*blockArea]
				d, _ := w.dct.Exec(data)
				s, _, err := w.svd.Exec(d)
				if err != nil {
					return
				}
				v := w.extract(s[0], s[1])
				setValue(at, v)
			}
		}(yuv)
	}
	wg.Wait()

	avrs := make([]float64, markLen)
	for i := range dist {
		avrs[i] = dist[i].Average()
	}
	return kmeans.OneDimKmeans(avrs), nil
}

func (w *Watermark) setEmbedD1(d1 int) error {
	fd1 := float64(d1)
	w.embed = func(s0, s1, bit float64) (r0 float64, r1 float64) {
		r0 = (float64(int(s0)/d1) + 1.0/4.0 + 1.0/2.0*0.5*bit) * fd1
		r1 = s1
		return
	}
	return nil
}

func (w *Watermark) setEmbedD1D2(d1, d2 int) error {
	fd1, fd2 := float64(d1), float64(d2)
	w.embed = func(s0, s1, bit float64) (r0 float64, r1 float64) {
		r0 = (float64(int(s0)/d1) + 1.0/4.0 + 1.0/2.0*0.5*bit) * fd1
		r1 = (float64(int(s1)/d2) + 1.0/4.0 + 1.0/2.0*0.5*bit) * fd2
		return
	}
	return nil
}

func (w *Watermark) setExtractD1(d1 int) error {
	w.extract = func(s0, s1 float64) (v float64) {
		if int(s0)%d1 > d1/2 {
			return 1
		}
		return 0
	}
	return nil
}

func (w *Watermark) setExtractD1D2(d1, d2 int) error {
	w.extract = func(s0, s1 float64) (v float64) {
		if int(s0)%d1 > d1/2 {
			v = 1
		}
		if int(s1)%d2 > d2/2 {
			return (v*3 + 1) / 4.
		}
		return (v * 3) / 4.
	}
	return nil
}

func (w *Watermark) init() error {
	defaultD1, defaultD2 := 36, 20
	defaultShape := [2]int{4, 4}
	if w.embed == nil || w.extract == nil {
		if err := w.setEmbedD1D2(defaultD1, defaultD2); err != nil {
			return err
		}
		if err := w.setExtractD1D2(defaultD1, defaultD2); err != nil {
			return err
		}
	}
	if w.dct == nil || w.svd == nil {
		w.blockShape = defaultShape
		w.dct = dct.New(defaultShape[0], defaultShape[1])
		w.svd = svd.New(defaultShape[0], defaultShape[1])
	}
	return nil
}
