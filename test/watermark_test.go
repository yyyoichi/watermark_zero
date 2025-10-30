package test

import (
	"context"
	"fmt"
	"image"
	"image/color"

	watermark "github.com/yyyoichi/watermark_zero"
)

func ExampleNew() {
	// Create a simple gradient image (100x100 pixels)
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			// Create gradient effect: red increases with x, green increases with y, blue is a mix
			r := uint8(x * 255 / 100)
			g := uint8(y * 255 / 100)
			b := uint8((x + y) * 255 / 200)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Initialize watermark processor with default settings
	w, err := watermark.New()
	if err != nil {
		fmt.Printf("Error creating watermark: %v\n", err)
		return
	}

	// Define a bit sequence to embed
	mark := []bool{true, false, true, true, false, false, true, false}

	// Embed the watermark
	ctx := context.Background()
	markedImg, err := w.Embed(ctx, img, mark)
	if err != nil {
		fmt.Printf("Error embedding watermark: %v\n", err)
		return
	}

	// Extract the watermark
	extractedMark, err := w.Extract(ctx, markedImg, len(mark))
	if err != nil {
		fmt.Printf("Error extracting watermark: %v\n", err)
		return
	}

	// Compare original and extracted marks
	fmt.Printf("Original:  %v\n", mark)
	fmt.Printf("Extracted: %v\n", extractedMark)

	// Check if they match
	match := true
	for i := range mark {
		if mark[i] != extractedMark[i] {
			match = false
			break
		}
	}
	fmt.Printf("Match: %v\n", match)

	// Output:
	// Original:  [true false true true false false true false]
	// Extracted: [true false true true false false true false]
	// Match: true
}
