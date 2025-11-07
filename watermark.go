package watermark

import (
	"context"
	"errors"
	"fmt"
	"image"

	"github.com/yyyoichi/watermark_zero/internal/dct"
	"github.com/yyyoichi/watermark_zero/internal/watermark"
)

var (
	ErrTooSmallImage = errors.New("image is too small for embedding or extracting")
)

type Watermark struct {
	d1, d2     int
	blockShape watermark.BlockShape
}

// New initializes a watermark processing structure.
// The blockShape and watermark coefficients d1, d2 can be optionally specified.
// For default values, refer to the init function.
func New(opts ...Option) (*Watermark, error) {
	w := new(Watermark)
	if err := w.init(opts...); err != nil {
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
	img := watermark.NewImageCore(src)
	if err := watermark.Enable(img, len(mark), w.blockShape); err != nil {
		return nil, fmt.Errorf("%w:%w", ErrTooSmallImage, err)
	}
	return watermark.Embed(ctx, img, mark, w.blockShape, w.d1, w.d2, nil, nil)
}

// BatchEmbed returns a function that can embed watermarks multiple times into the same source image efficiently.
func (w *Watermark) BatchEmbed(src image.Image) func(ctx context.Context, mark []bool, opts ...Option) (image.Image, error) {
	var (
		original = watermark.NewImageCore(src)
		wavelets = watermark.Wavelets(original)
		dctCache = dct.NewCache()
	)

	var fn = func(ctx context.Context, mark []bool, opts ...Option) (image.Image, error) {
		w, err := w.add(opts...)
		if err != nil {
			return nil, err
		}
		img := original.Copy()
		if err := watermark.Enable(img, len(mark), w.blockShape); err != nil {
			return nil, fmt.Errorf("%w:%w", ErrTooSmallImage, err)
		}
		return watermark.Embed(ctx, img, mark, w.blockShape, w.d1, w.d2, wavelets, dctCache)
	}
	return fn
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
	img := watermark.NewImageCore(src)
	if err := watermark.Enable(img, markLen, w.blockShape); err != nil {
		return nil, fmt.Errorf("%w:%w", ErrTooSmallImage, err)
	}
	return watermark.Extract(ctx, img, markLen, w.blockShape, w.d1, w.d2, nil, nil)
}

// BatchExtract returns a function that can extract watermarks multiple times from the same source image efficiently.
func (w *Watermark) BatchExtract(src image.Image) func(ctx context.Context, markLen int, opts ...Option) ([]bool, error) {
	var (
		original = watermark.NewImageCore(src)
		wavelets = watermark.Wavelets(original)
		dctCache = dct.NewCache()
	)

	var fn = func(ctx context.Context, markLen int, opts ...Option) ([]bool, error) {
		w, err := w.add(opts...)
		if err != nil {
			return nil, err
		}
		img := original.Copy()
		if err := watermark.Enable(img, markLen, w.blockShape); err != nil {
			return nil, fmt.Errorf("%w:%w", ErrTooSmallImage, err)
		}
		return watermark.Extract(ctx, img, markLen, w.blockShape, w.d1, w.d2, wavelets, dctCache)
	}
	return fn
}

func (w *Watermark) add(opts ...Option) (*Watermark, error) {
	var copy = Watermark{
		d1:         w.d1,
		d2:         w.d2,
		blockShape: w.blockShape,
	}
	if err := copy.init(opts...); err != nil {
		return nil, err
	}
	return &copy, nil
}

func (w *Watermark) init(opts ...Option) error {
	for _, opt := range opts {
		if err := opt(w); err != nil {
			return err
		}
	}
	if w.d1 == 0 {
		w.d1 = 36
		w.d2 = 20
	}
	if w.blockShape.IsZero() {
		w.blockShape = watermark.NewBlockShape(8, 8)
	}
	return nil
}
