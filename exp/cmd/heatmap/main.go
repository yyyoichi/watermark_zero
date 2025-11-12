package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"time"

	images "exp/internal/images"
	markpkg "exp/internal/mark"

	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/strmark/wzeromark"
)

// This tool creates heatmap PNGs showing positions of mismatched encoded bits after
// embedding+JPEG compression+extraction. It saves outputs under /tmp/heatmap/.

func main() {
	idx := flag.Int("i", 10, "image index to process (0-based)")
	outDir := flag.String("out", "/tmp/heatmap", "output directory")
	flag.Parse()

	// Parameters (can be expanded)
	imageSizes := [][]int{{426, 240}}
	blockShapes := [][]int{{6, 6}}
	d1d2Pairs := [][]int{{25, 14}}

	urls := images.ParseURLs()
	if len(urls) == 0 {
		log.Fatal("no image URLs available in images package")
	}
	if *idx < 0 || *idx >= len(urls) {
		log.Fatalf("image index %d out of range (0..%d)", *idx, len(urls)-1)
	}
	url := urls[*idx]

	// Prepare watermark mark (Golay)
	seed := make([]byte, 32)
	_ = seed // deterministic empty seed is OK for experiments; user can change
	m, err := wzeromark.New(seed, seed, "1a2b")
	if err != nil {
		log.Fatalf("failed to create watermark: %v", err)
	}
	testMark, err := m.Encode("TEST_MARK")
	if err != nil {
		log.Fatalf("failed to encode test mark: %v", err)
	}
	golayMark := markpkg.NewGolayMark(testMark)

	// Ensure output dir
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("failed to create out dir: %v", err)
	}

	ctx := context.Background()

	for _, size := range imageSizes {
		for _, bs := range blockShapes {
			for _, d1d2 := range d1d2Pairs {
				width, height := size[0], size[1]
				img, err := images.FetchImageWithSize(url, width, height)
				if err != nil {
					log.Printf("error fetching image: %v", err)
					continue
				}

				// Embed
				opts := []watermark.Option{
					watermark.WithBlockShape(bs[1], bs[0]), // width, height
					watermark.WithD1D2(d1d2[0], d1d2[1]),
				}
				batch := watermark.NewBatch(img)
				markedImg, err := batch.Embed(ctx, golayMark.Encoded, opts...)
				if err != nil {
					log.Printf("embed error: %v", err)
					continue
				}

				// JPEG compress/decode
				var buf bytes.Buffer
				if err := jpeg.Encode(&buf, markedImg, &jpeg.Options{Quality: 100}); err != nil {
					log.Printf("jpeg encode error: %v", err)
					continue
				}
				compressedImg, err := jpeg.Decode(&buf)
				if err != nil {
					log.Printf("jpeg decode error: %v", err)
					continue
				}

				// Extract
				extracted, err := watermark.Extract(ctx, compressedImg, len(golayMark.Encoded), opts...)
				if err != nil {
					log.Printf("extract error: %v", err)
					continue
				}

				// Compare encoded vs extracted and build heatmap overlay
				// (lengths of encoded and extracted are guaranteed to match)
				mismatches := make(map[int]bool)
				for i := range golayMark.Encoded {
					if golayMark.Encoded[i] == extracted[i] {
						continue
					}
					mismatches[i] = true
				}

				// Decode extracted bits back to original and check decode success per original block
				decoded := golayMark.Decode(extracted)
				decodedMatches := 0
				for i := range golayMark.Original {
					if golayMark.Original[i] == decoded[i] {
						decodedMatches++
					}
				}
				decodedAccuracy := float64(decodedMatches) / float64(len(golayMark.Original)) * 100

				// original block size for Golay is 12 bits
				originalBlockSize := 12
				// (664+ 11) / 12 = 56 blocks
				numOriginalBlocks := (len(golayMark.Original) + originalBlockSize - 1) / originalBlockSize
				failedOriginalBlocks := make([]bool, numOriginalBlocks)
				for i := range golayMark.Original {
					if golayMark.Original[i] == decoded[i] {
						continue
					}
					blockIdx := i / originalBlockSize
					failedOriginalBlocks[blockIdx] = true
				}

				// Build overlay image
				out := image.NewRGBA(markedImg.Bounds())
				draw.Draw(out, out.Bounds(), markedImg, image.Point{}, draw.Src)

				rect := markedImg.Bounds()
				blocksPerRow := (rect.Dx() + 1) / bs[1]
				blocksPerCol := (rect.Dy() + 1) / bs[0]
				totalBlocks := blocksPerRow * blocksPerCol

				// For each image block, decide overlay color by:
				// - if the corresponding encoded block failed to decode to the original -> PINK (priority)
				// - else if the encoded bit at this position mismatched -> BLUE
				for b := 0; b < totalBlocks; b++ {
					encIdx := b % len(golayMark.Encoded)
					row := b / blocksPerRow
					col := b % blocksPerRow
					x0 := col * bs[1]
					y0 := row * bs[0]
					x1 := x0 + bs[1]
					y1 := y0 + bs[0]
					if x1 > rect.Dx() {
						x1 = rect.Dx()
					}
					if y1 > rect.Dy() {
						y1 = rect.Dy()
					}

					encBlockIdx := (encIdx / 23) % numOriginalBlocks
					// If this encoded block corresponds to a decode-failed original block, paint pink first
					if encBlockIdx < len(failedOriginalBlocks) && failedOriginalBlocks[encBlockIdx] {
						// blend pink over the rectangle per-pixel for consistent visibility
						pink := color.RGBA{R: 255, G: 105, B: 180, A: 200}
						blendRect(out, image.Rect(x0, y0, x1, y1), pink)
					}

					// Then, if this encoded position mismatched, paint blue on top (so blue shows over pink)
					if mismatches[encIdx] {
						blue := color.RGBA{R: 0, G: 0, B: 255, A: 200}
						blendRect(out, image.Rect(x0, y0, x1, y1), blue)
					}
				}

				// Count mismatches and failed original blocks for logging
				encodedMismatchCount := 0
				for _, v := range mismatches {
					if v {
						encodedMismatchCount++
					}
				}
				failedOriginalBlocksCount := 0
				for _, v := range failedOriginalBlocks {
					if v {
						failedOriginalBlocksCount++
					}
				}

				// Save PNG
				fname := fmt.Sprintf("img%03d_%dx%d_bs%dx%d_d%dx%d_decAcc%02.0f_encFail%03d_decFail%02d.png", *idx, width, height, bs[1], bs[0], d1d2[0], d1d2[1], decodedAccuracy, encodedMismatchCount, failedOriginalBlocksCount)
				outPath := filepath.Join(*outDir, fname)
				f, err := os.Create(outPath)
				if err != nil {
					log.Printf("failed to create out file: %v", err)
					continue
				}
				if err := png.Encode(f, out); err != nil {
					log.Printf("failed to encode png: %v", err)
				}
				_ = f.Close()
				log.Printf("wrote %s (encoded bit mismatches: %d, failed original blocks: %d, decodedAcc=%.2f%%)\n", outPath, encodedMismatchCount, failedOriginalBlocksCount, decodedAccuracy)

				time.Sleep(time.Duration(time.Millisecond * 200))
			}
		}
	}
}

// blendRect blends a semi-opaque overlay color into dst for every pixel inside r.
// dst must be *image.RGBA.
func blendRect(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	// clamp rectangle to dst bounds
	r = r.Intersect(dst.Bounds())
	if r.Empty() {
		return
	}
	a := float64(c.A) / 255.0
	invA := 1.0 - a
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			i := dst.PixOffset(x, y)
			or := float64(dst.Pix[i+0])
			og := float64(dst.Pix[i+1])
			ob := float64(dst.Pix[i+2])
			oa := float64(dst.Pix[i+3]) / 255.0

			nr := uint8(a*float64(c.R) + invA*or)
			ng := uint8(a*float64(c.G) + invA*og)
			nb := uint8(a*float64(c.B) + invA*ob)
			// new alpha keep as fully opaque (preserve original alpha)
			na := uint8((oa) * 255)

			dst.Pix[i+0] = nr
			dst.Pix[i+1] = ng
			dst.Pix[i+2] = nb
			dst.Pix[i+3] = na
		}
	}
}
