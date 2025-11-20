package watermark_test

import (
	"context"
	"fmt"
	"image"

	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/mark"
)

func Example_watermark() {
	ctx := context.Background()
	img := image.NewGray(image.Rect(0, 0, 200, 200))
	// Initialize watermark processor with default settings
	w, _ := watermark.New(
		watermark.WithBlockShape(4, 6),
		watermark.WithD1D2(21, 11),
	)

	// Define a bit sequence to embed
	mark := mark.NewString("Test-Mark")

	// Embed the watermark
	markedImg, _ := w.Embed(ctx, img, mark)

	// Extract the watermark
	extractedMark, _ := w.Extract(ctx, markedImg, mark)
	fmt.Println(extractedMark.DecodeToString())

	// Output:
	// Test-Mark
}

func Example_batch() {
	ctx := context.Background()
	img := image.NewGray(image.Rect(0, 0, 200, 200))

	opts := []watermark.Option{
		watermark.WithBlockShape(4, 4),
		watermark.WithD1D2(21, 11),
	}

	batch := watermark.NewBatch(img)
	for _, m := range []string{"Hello!", "こんにちは！"} {
		mark := mark.NewString(m)
		markedImg, _ := batch.Embed(ctx, mark, opts...)

		extractedMark, _ := watermark.Extract(ctx, markedImg, mark, opts...)

		fmt.Println(extractedMark.DecodeToString())
	}

	// Output:
	// Hello!
	// こんにちは！
}
