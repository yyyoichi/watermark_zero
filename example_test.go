package watermark_test

import (
	"context"
	"fmt"
	"image"
	"image/color"

	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/strmark"
)

func Example_watermark() {
	// Create a simple gradient image (200x200 pixels)
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
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
	w, err := watermark.New(
		watermark.WithBlockShape(4, 6),
		watermark.WithD1D2(36, 20),
	)
	if err != nil {
		fmt.Printf("Error creating watermark: %v\n", err)
		return
	}

	// Define a bit sequence to embed
	mark := strmark.Encode("Test-Mark")

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

	fmt.Println(strmark.Decode(extractedMark))

	// Output:
	// Test-Mark
}
