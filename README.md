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

## Benchmark

```txt
$ go test -bench=. ./bench/ -bench=^BenchmarkEmbed_FHD$ -benchmem -cpu=1,2,3,4 -benchtime=10s
goos: linux
goarch: amd64
pkg: github.com/yyyoichi/watermark_zero/bench
cpu: Intel(R) Core(TM) i5-8250U CPU @ 1.60GHz
BenchmarkEmbed_FHD/4x4_D1                     18         683055045 ns/op        708325051 B/op  13349116 allocs/op
BenchmarkEmbed_FHD/4x4_D1-2                   12         931026118 ns/op        708289144 B/op  13349014 allocs/op
BenchmarkEmbed_FHD/4x4_D1-3                   18         732031677 ns/op        708301659 B/op  13349061 allocs/op
BenchmarkEmbed_FHD/4x4_D1-4                   15         767310390 ns/op        708324182 B/op  13349109 allocs/op
BenchmarkEmbed_FHD/4x4_D1D2                   14         774766221 ns/op        708326742 B/op  13349116 allocs/op
BenchmarkEmbed_FHD/4x4_D1D2-2                 12         912351192 ns/op        708287454 B/op  13349013 allocs/op
BenchmarkEmbed_FHD/4x4_D1D2-3                 16         750195511 ns/op        708301917 B/op  13349062 allocs/op
BenchmarkEmbed_FHD/4x4_D1D2-4                 14         783641077 ns/op        708324888 B/op  13349112 allocs/op
BenchmarkEmbed_FHD/8x8_D1                     24         454617080 ns/op        373166599 B/op   4892570 allocs/op
BenchmarkEmbed_FHD/8x8_D1-2                   21         533784334 ns/op        373136603 B/op   4892521 allocs/op
BenchmarkEmbed_FHD/8x8_D1-3                   28         443597994 ns/op        373147044 B/op   4892542 allocs/op
BenchmarkEmbed_FHD/8x8_D1-4                   26         443126212 ns/op        373163018 B/op   4892566 allocs/op
BenchmarkEmbed_FHD/8x8_D1D2                   26         462500377 ns/op        373164116 B/op   4892566 allocs/op
BenchmarkEmbed_FHD/8x8_D1D2-2                 21         548263726 ns/op        373137639 B/op   4892518 allocs/op
BenchmarkEmbed_FHD/8x8_D1D2-3                 25         459467989 ns/op        373150312 B/op   4892545 allocs/op
BenchmarkEmbed_FHD/8x8_D1D2-4                 26         441787512 ns/op        373163220 B/op   4892566 allocs/op
BenchmarkEmbed_FHD/12x12_D1                   24         459372212 ns/op        310172016 B/op   3326536 allocs/op
BenchmarkEmbed_FHD/12x12_D1-2                 21         530219781 ns/op        310152254 B/op   3326507 allocs/op
BenchmarkEmbed_FHD/12x12_D1-3                 27         442878278 ns/op        310159773 B/op   3326520 allocs/op
BenchmarkEmbed_FHD/12x12_D1-4                 25         461542104 ns/op        310172684 B/op   3326534 allocs/op
BenchmarkEmbed_FHD/12x12_D1D2                 25         462859746 ns/op        310172977 B/op   3326538 allocs/op
BenchmarkEmbed_FHD/12x12_D1D2-2               21         539403559 ns/op        310151166 B/op   3326505 allocs/op
BenchmarkEmbed_FHD/12x12_D1D2-3               26         447806862 ns/op        310160117 B/op   3326522 allocs/op
BenchmarkEmbed_FHD/12x12_D1D2-4               27         463405114 ns/op        310173060 B/op   3326537 allocs/op
BenchmarkEmbed_FHD/16x16_D1                   26         467656523 ns/op        286880552 B/op   2773204 allocs/op
BenchmarkEmbed_FHD/16x16_D1-2                 20         547519096 ns/op        286844256 B/op   2773178 allocs/op
BenchmarkEmbed_FHD/16x16_D1-3                 24         465889750 ns/op        286859918 B/op   2773190 allocs/op
BenchmarkEmbed_FHD/16x16_D1-4                 25         461271046 ns/op        286878939 B/op   2773205 allocs/op
BenchmarkEmbed_FHD/16x16_D1D2                 24         458887882 ns/op        286877920 B/op   2773204 allocs/op
BenchmarkEmbed_FHD/16x16_D1D2-2               20         540008836 ns/op        286843490 B/op   2773176 allocs/op
BenchmarkEmbed_FHD/16x16_D1D2-3               25         457357105 ns/op        286855818 B/op   2773190 allocs/op
BenchmarkEmbed_FHD/16x16_D1D2-4               24         463504708 ns/op        286878187 B/op   2773204 allocs/op
PASS
ok      github.com/yyyoichi/watermark_zero/bench        368.540s
```