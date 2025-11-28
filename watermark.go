package watermark

import (
	"context"
	"errors"
	"fmt"
	"image"

	"github.com/yyyoichi/watermark_zero/internal/dct"
	"github.com/yyyoichi/watermark_zero/internal/dwt"
	"github.com/yyyoichi/watermark_zero/internal/watermark"
)

var (
	ErrTooSmallImage = errors.New("image is too small for embedding or extracting")
)

// Embed embeds a bit sequence into an image with the specified options.
// This is a convenience function that creates a Watermark instance and calls its Embed method.
func Embed(ctx context.Context, src image.Image, mark EmbedMark, opts ...Option) (image.Image, error) {
	w, _ := New(opts...)
	return w.Embed(ctx, src, mark)
}

// Extract extracts a bit sequence from an image with the specified options.
// This is a convenience function that creates a Watermark instance and calls its Extract method.
func Extract(ctx context.Context, src image.Image, mark ExtractMark, opts ...Option) (MarkDecoder, error) {
	w, _ := New(opts...)
	return w.Extract(ctx, src, mark)
}

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
func (w *Watermark) Embed(ctx context.Context, src image.Image, mark EmbedMark) (image.Image, error) {
	img := watermark.NewImageCore(src)
	if err := watermark.Enable(img, mark.Len(), w.blockShape); err != nil {
		return nil, fmt.Errorf("%w:%w", ErrTooSmallImage, err)
	}
	return watermark.Embed(ctx, img, mark, w.blockShape, w.d1, w.d2, nil, nil)
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
func (w *Watermark) Extract(ctx context.Context, src image.Image, mark ExtractMark) (MarkDecoder, error) {
	img := watermark.NewImageCore(src)
	if err := watermark.Enable(img, mark.Len(), w.blockShape); err != nil {
		return nil, fmt.Errorf("%w:%w", ErrTooSmallImage, err)
	}
	bits, err := watermark.Extract(ctx, img, mark.Len(), w.blockShape, w.d1, w.d2, nil, nil)
	if err != nil {
		return nil, err
	}
	return mark.NewDecoder(bits), nil
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

// Batch enables efficient multiple watermark operations on a single image
// by caching intermediate computation results (wavelets and DCT).
type Batch struct {
	original watermark.ImageSource
	wavelets []*dwt.Wavelets
	dctCache *dct.Cache
}

// NewBatch creates a new Batch instance and pre-computes wavelet transforms
// and initializes DCT cache for the given image.
func NewBatch(src image.Image) *Batch {
	b := &Batch{
		original: watermark.NewImageCore(src),
		dctCache: dct.NewCache(),
	}
	b.wavelets = watermark.Wavelets(b.original)
	return b
}

// Embed embeds a bit sequence into the cached image with specified options.
func (b *Batch) Embed(ctx context.Context, mark EmbedMark, opts ...Option) (image.Image, error) {
	w, _ := New(opts...)
	img := b.original.Copy()
	if err := watermark.Enable(img, mark.Len(), w.blockShape); err != nil {
		return nil, fmt.Errorf("%w:%w", ErrTooSmallImage, err)
	}
	// Uses pre-computed wavelets and DCT cache for improved performance.
	return watermark.Embed(ctx, img, mark, w.blockShape, w.d1, w.d2, b.wavelets, b.dctCache)
}

// Extract extracts a bit sequence from the cached image with specified options.
func (b *Batch) Extract(ctx context.Context, markLen int, opts ...Option) ([]byte, error) {
	w, _ := New(opts...)
	img := b.original.Copy()
	if err := watermark.Enable(img, markLen, w.blockShape); err != nil {
		return nil, fmt.Errorf("%w:%w", ErrTooSmallImage, err)
	}
	// Uses pre-computed wavelets and DCT cache for improved performance.
	return watermark.Extract(ctx, img, markLen, w.blockShape, w.d1, w.d2, b.wavelets, b.dctCache)
}
