package watermark_test

import (
	"context"
	"fmt"
	"image"

	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/strmark"
)

func Example_watermark() {
	ctx := context.Background()
	img := image.NewGray(image.Rect(0, 0, 200, 200))
	// Initialize watermark processor with default settings
	w, _ := watermark.New(
		watermark.WithBlockShape(4, 6),
		watermark.WithD1D2(36, 20),
	)

	// Define a bit sequence to embed
	mark := watermark.NewBoolEmbedMark(strmark.Encode("Test-Mark"))

	// Embed the watermark
	markedImg, _ := w.Embed(ctx, img, mark)

	// Extract the watermark
	extractedMark, _ := w.Extract(ctx, markedImg, mark.Len())
	fmt.Println(strmark.Decode(extractedMark))

	// Output:
	// Test-Mark
}

func Example_batch() {
	ctx := context.Background()
	img := image.NewGray(image.Rect(0, 0, 200, 200))

	opts := []watermark.Option{
		watermark.WithBlockShape(4, 4),
		watermark.WithD1D2(32, 18),
	}

	batch := watermark.NewBatch(img)
	for _, m := range []string{"Hello!", "こんにちは！"} {
		mark := watermark.NewBoolEmbedMark(strmark.Encode(m))
		markedImg, _ := batch.Embed(ctx, mark, opts...)

		extractedMark, _ := watermark.Extract(ctx, markedImg, mark.Len(), opts...)

		fmt.Println(strmark.Decode(extractedMark))
	}

	// Output:
	// Hello!
	// こんにちは！
}
