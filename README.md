# watermark_zero

The High-Efficiency Go Implementation of guofei9987/blind_watermark

> This Go language implementation draws inspiration from the ideas and approach of [guofei9987/blind_watermark](https://github.com/guofei9987/blind_watermark) regarding the fundamental signal processing logic for digital watermarks.
> The source of the base logic is the MIT License, and its copyright notice and license terms are included in the THIRD_PARTY_LICENSES.txt file within this repository.


## Install

`go get github.com/yyyoichi/watermark_zero`

## Example

```golang

func ExampleNew() {
	// Create a simple gradient image (100x100 pixels)
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			r := uint8(x * 255 / 100)
			g := uint8(y * 255 / 100)
			b := uint8((x + y) * 255 / 200)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// Initialize watermark processor with default settings
	w, _ := watermark.New()
	// Define a bit sequence to embed
	mark := []bool{true, false, true, true, false, false, true, false}
    // Embed the watermark
	markedImg, _ := w.Embed(context.Background(), img, mark)
	// Extract the watermark
	extractedMark, _ := w.Extract(ctx, markedImg, len(mark))

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

```