# watermark_zero

The High-Efficiency Go Implementation of guofei9987/blind_watermark

> This Go language implementation draws inspiration from the ideas and approach of [guofei9987/blind_watermark](https://github.com/guofei9987/blind_watermark) regarding the fundamental signal processing logic for digital watermarks.
> The source of the base logic is the MIT License, and its copyright notice and license terms are included in the THIRD_PARTY_LICENSES.txt file within this repository.


## Install

`go get github.com/yyyoichi/watermark_zero`

## Example

```golang

func main() {
	var img image.Image // load from anywhere 
	w, _ := watermark.New(
		watermark.WithBlockShape(8, 6),
		watermark.WithD1D2(36, 20),
	)
	mark := mark.NewString("@Copyright2025_USERXX")
	markedImg, _ := w.Embed(context.Background(), img, mark)
}

```

## `WithoutECC` vs `WithGolay`

This section provides a comparison and explanation of mark encoding methods.

`WithoutECC` uses traditional byte arrays converted directly into bit sequences for embedding.
For example, `a` would be converted as: `0x61` -> `0b01100001`

`WithGolay` utilizes Golay code (23,12) with error correction capability. Golay code is an error-correcting code that encodes 12 bits of data into 23 bits, enabling correction of up to 3-bit errors.
For example, when a 12-bit sequence is Golay-encoded, it becomes a 23-bit sequence.

In conclusion, **we strongly recommend using `WithGolay`**.

### Characteristics in Digital Watermarking

In digital watermark extraction, success is only achieved when all bits of the bit sequence are correctly extracted.
In other words, if even 1 bit is incorrect, the entire process is considered a failure. For example, if you embed 100 bits and correctly extract 50 bits, 1 bit, or even 99 bits, anything less than 100 bits is considered a complete failure.

With `WithoutECC`, since not even a single bit can be incorrect, *the extraction success rate is likely to be lower*.

With `WithGolay`, although nearly twice the amount of information is embedded, error correction allows for tolerance of some errors while still achieving success, meaning *it is expected to increase the extraction success rate*.

### Comparison

![sgolay-vs-noecc](./docs/images/compare-sgolay.png)

#### Reference: Golay Encoding Package

https://pkg.go.dev/github.com/yyyoichi/golay

```
go get github.com/yyyoichi/golay
```


## Parameters and Noise Level

Increasing the D1 and D2 parameters improves the extraction success rate, but image quality will degrade.
Additionally, dividing the image into finer blocks increases the embedding capacity, but also increases noise.

![ssim](./docs/images/ssim.png)

![success-rate](./docs/images/success-rate.png)

## Reference

These examples are implemented in [./exp](./exp/).
All experiments were conducted to investigate resistance to JPEG compression.

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
