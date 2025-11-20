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
		watermark.WithBlockShape(4, 4),
		watermark.WithD1D2(21, 11),
	)

	// Define a bit sequence to embed
	m := mark.NewString("Test-Mark")

	// Embed the watermark
	markedImg, _ := w.Embed(ctx, img, m)

	// Extract the watermark
	extractedMark, _ := w.Extract(ctx, markedImg, m)
	fmt.Println(extractedMark.DecodeToString())
	exM := mark.NewExtract(m.ExtractSize())
	extractedMark, _ = w.Extract(ctx, markedImg, exM)
	fmt.Println(extractedMark.DecodeToString())

	// Output:
	// Test-Mark
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

func Example_mismatchedMarkOptions() {
	ctx := context.Background()
	img := image.NewGray(image.Rect(0, 0, 200, 200))
	w, _ := watermark.New(
		watermark.WithBlockShape(4, 4),
		watermark.WithD1D2(21, 11),
	)

	// Embed a mark with Golay encoding (default seed)
	embedMark := mark.NewString("Test")
	markedImg, _ := w.Embed(ctx, img, embedMark)

	// Try to extract with different options (WithoutECC)
	wrongExtractMark1 := mark.NewExtract(embedMark.ExtractSize(), mark.WithoutECC())
	wrongDecoded1, _ := w.Extract(ctx, markedImg, wrongExtractMark1)
	result1 := wrongDecoded1.DecodeToString()
	fmt.Printf("WithoutECC matches 'Test': %v\n", result1 == "Test")

	// Try to extract with different seed
	wrongExtractMark2 := mark.NewExtract(embedMark.ExtractSize(), mark.WithGolay(99999))
	wrongDecoded2, _ := w.Extract(ctx, markedImg, wrongExtractMark2)
	result2 := wrongDecoded2.DecodeToString()
	fmt.Printf("Different seed matches 'Test': %v\n", result2 == "Test")

	// Extract with correct options (same as embedding)
	correctExtractMark := mark.NewExtract(embedMark.ExtractSize()) // Uses default Golay encoding and default seed
	correctDecoded, _ := w.Extract(ctx, markedImg, correctExtractMark)
	result3 := correctDecoded.DecodeToString()
	fmt.Printf("Correct options matches 'Test': %v\n", result3 == "Test")

	// Output:
	// WithoutECC matches 'Test': false
	// Different seed matches 'Test': false
	// Correct options matches 'Test': true
}
