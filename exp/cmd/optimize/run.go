package main

import (
	"bytes"
	"context"
	"encoding/json"
	"exp/internal/images"
	"exp/internal/mark"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/yyyoichi/bitstream-go"
	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/strmark/wzeromark"
)

var TEST_MARK = func() []bool {
	w := bitstream.NewBitWriter[uint64](0, 0)
	for i := range wzeromark.MarkLen / 8 {
		w.U8(0, 8, uint8(i*2))
	}
	d, _ := w.Data()
	r := bitstream.NewBitReader(d, 0, 0)
	var data = make([]bool, wzeromark.MarkLen)
	for i := range data {
		data[i] = r.U8R(1, i) == 1
	}
	return data
}()

func runMain(numImages, offset int, targetEmbedLow, targetEmbedHigh float64) {
	ctx := context.Background()

	// Generate image sizes focusing on EmbedCount 1~8 (threshold around 5)
	// EmbedCount = TotalBlocks / EncodedBits
	// For ShuffledGolay: EncodedBits = OriginalBits * 23/12 ≈ 1.92x
	// With 8x8 blocks: EmbedCount ≈ (width * height) / 82432
	imageSizes := [][]int{
		// EmbedCount 0.5-1
		{320, 180}, // ~0.79 (8x8)
		// EmbedCount 1-2
		{384, 216}, // ~1.01 (8x8)
		{426, 240}, // ~1.24 (8x8)
		{480, 270}, // ~1.57 (8x8)
		{512, 288}, // ~1.79 (8x8)
		// EmbedCount 2-4
		{600, 338}, // ~2.46 (8x8)
		{640, 360}, // ~2.79 (8x8)
		{720, 405}, // ~3.53 (8x8)
		{768, 432}, // ~4.02 (8x8)
		// EmbedCount 4-6 (threshold area)
		{800, 450}, // ~4.36 (8x8)
		{854, 480}, // ~4.97 (8x8)
		{896, 504}, // ~5.47 (8x8)
		{960, 540}, // ~6.28 (8x8)
	}

	// Block shape: Fixed to 8x8 only for focused analysis
	blockShapes := [][]int{
		{8, 8},
		{6, 6},
		{4, 4},
	}

	// D1/D2 parameter space for optimization
	d1d2Pairs := [][]int{
		{21, 11},
		{21, 9},
		{21, 7},
		{21, 5},
		{21, 3},
		{19, 11},
		{19, 9},
		{19, 7},
		{19, 5},
		{19, 3},
		{17, 11},
		{17, 9},
		{17, 7},
		{17, 5},
		{17, 3},
		{15, 11},
		{15, 9},
		{15, 7},
		{15, 5},
		{15, 3},
	} // Parsse image URLs
	urls := images.ParseURLs()
	if len(urls) == 0 {
		log.Fatal("No image URLs found")
	}

	// Apply offset and limit
	if offset >= len(urls) {
		log.Fatalf("Offset %d is beyond available images (%d)", offset, len(urls))
	}
	urls = urls[offset:]
	if numImages > 0 && numImages < len(urls) {
		urls = urls[:numImages]
	}

	shuffledGolay := mark.NewShuffledGolayMark(TEST_MARK)

	log.Printf("Starting D1/D2 optimization with %d images (offset=%d)\n", len(urls), offset)
	log.Printf("Total test cases per image: %d (image sizes) x %d (block shapes) x %d (d1/d2 pairs) = %d\n",
		len(imageSizes), len(blockShapes), len(d1d2Pairs), len(imageSizes)*len(blockShapes)*len(d1d2Pairs))

	var allResults []OptimizeResult

	for i, url := range urls {
		log.Printf("\n[%d/%d] Testing image: %s\n", i+1, len(urls), url)

		for _, size := range imageSizes {
			width, height := size[0], size[1]
			sizeKey := fmt.Sprintf("%dx%d", width, height)
			log.Printf("  Size: %s\n", sizeKey)

			img, err := images.FetchImageWithSize(url, width, height)
			if err != nil {
				log.Printf("    Error fetching image: %v\n", err)
				continue
			}

			batch := watermark.NewBatch(img)
			rect := img.Bounds()

			// Pre-calculate test parameters that pass the filter
			var testParams []TestParams
			for _, bs := range blockShapes {
				for _, d1d2 := range d1d2Pairs {
					totalBlocks := ((rect.Dx() + 1) / bs[1]) * ((rect.Dy() + 1) / bs[0])
					embedCount := float64(totalBlocks) / float64(len(shuffledGolay.Encoded))

					if embedCount < targetEmbedLow || embedCount > targetEmbedHigh {
						continue
					}

					testParams = append(testParams, TestParams{
						BlockShapeH:       bs[0],
						BlockShapeW:       bs[1],
						D1:                d1d2[0],
						D2:                d1d2[1],
						Mark:              shuffledGolay,
						TotalBlocks:       totalBlocks,
						ImageWidth:        width,
						ImageHeight:       height,
						EmbedCount:        embedCount,
						ImageName:         fmt.Sprintf("%03d", i+offset),
						OriginalImagePath: images.GetCachedImagePath(url, width, height),
					})
				}
			}

			if len(testParams) == 0 {
				log.Printf("    No tests to run for this size (all filtered out)\n")
				continue
			}

			// Create channels
			numWorkers := runtime.GOMAXPROCS(0)
			testParamsCh := make(chan TestParams, numWorkers)
			resultCh := make(chan *TestResult, len(testParams))

			// Start worker goroutines
			var wg sync.WaitGroup
			wg.Add(numWorkers)
			for range numWorkers {
				go func() {
					defer wg.Done()
					for params := range testParamsCh {
						result := testWatermark(ctx, batch, params)
						resultCh <- result
					}
				}()
			}
			go func() {
				defer close(resultCh)
				wg.Wait()
			}()

			// Send test parameters
			go func() {
				defer close(testParamsCh)
				for _, params := range testParams {
					testParamsCh <- params
				}
			}()

			// Collect results
			for result := range resultCh {
				if result == nil {
					continue
				}
				params := result.TestParams

				allResults = append(allResults, OptimizeResult{
					EmbedImagePath: params.EmbeddedImagePath(TmpOptimizeEmbeddedImagesDir),

					ImageSize:       sizeKey,
					ImageWidth:      params.ImageWidth,
					ImageHeight:     params.ImageHeight,
					BlockShapeW:     params.BlockShapeW,
					BlockShapeH:     params.BlockShapeH,
					D1:              params.D1,
					D2:              params.D2,
					EmbedCount:      params.EmbedCount,
					TotalBlocks:     params.TotalBlocks,
					EncodedAccuracy: result.EncodedAccuracy,
					DecodedAccuracy: result.DecodedAccuracy,
					Success:         result.Success,
					SSIM:            result.SSIM,
				})
			}
		}
	}

	log.Printf("\n=== Optimization Complete ===\n")
	log.Printf("Total test results: %d\n", len(allResults))
	log.Printf("Generating visualizations...\n")

	// Generate visualizations
	outDir := TmpOptimizeJsonsDir
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	// save as JSON and generate plots
	// DataJsonFormat
	var data = DataJsonFormat{
		Results: allResults,
	}
	data.Params.ImageSizes = imageSizes
	data.Params.BlockShapes = blockShapes
	data.Params.D1D2Pairs = d1d2Pairs
	data.Params.NumImages = len(urls)
	data.Params.Offset = offset
	data.Params.TargetEmbedLow = targetEmbedLow
	data.Params.TargetEmbedHigh = targetEmbedHigh

	filename := fmt.Sprintf("optimize_results_%d_images_offset_%d_ec-%.1f-%.1f.json", len(urls), offset, targetEmbedLow, targetEmbedHigh)
	f, err := os.Create(filepath.Join(outDir, filename))
	if err != nil {
		log.Fatalf("Failed to create JSON file: %v", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(data); err != nil {
		log.Fatalf("Failed to encode JSON data: %v", err)
	}

	log.Printf("\nResults saved to: %s\n", outDir)
	log.Println(filepath.Join(outDir, filename))
}

// TestParams holds parameters for a single test
type TestParams struct {
	BlockShapeH       int
	BlockShapeW       int
	D1                int
	D2                int
	Mark              mark.Mark
	TotalBlocks       int
	ImageWidth        int
	ImageHeight       int
	EmbedCount        float64
	ImageName         string
	OriginalImagePath string
}

// TestResult holds the test outcome
type TestResult struct {
	TestParams      *TestParams
	EncodedAccuracy float64
	DecodedAccuracy float64
	Success         bool
	SSIM            float64
}

func testWatermark(ctx context.Context, batch *watermark.Batch, params TestParams) *TestResult {
	opts := []watermark.Option{
		watermark.WithBlockShape(params.BlockShapeW, params.BlockShapeH),
		watermark.WithD1D2(params.D1, params.D2),
	}

	start := time.Now()

	embeddedPath := params.EmbeddedImagePath(TmpOptimizeEmbeddedImagesDir)
	embededJpeg, err := getEmbedImage(embeddedPath)
	if err != nil {
		// Embed
		embeddedImg, err := batch.Embed(ctx, params.Mark.Encoded, opts...)
		if err != nil {
			log.Printf("    [FAIL] Size=%dx%d BS=%dx%d D1D2=%dx%d EC=%.2f - Embed error: %v\n",
				params.ImageWidth, params.ImageHeight, params.BlockShapeW, params.BlockShapeH,
				params.D1, params.D2, params.EmbedCount, err)
			return nil
		}

		// Save embedded image for caching
		embededJpeg, err = saveEmbedImage(embeddedPath, embeddedImg)
		if err != nil {
			log.Printf("    [WARN] Size=%dx%d BS=%dx%d D1D2=%dx%d EC=%.2f - Save embed cache error: %v\n",
				params.ImageWidth,
				params.ImageHeight, params.BlockShapeW, params.BlockShapeH,
				params.D1, params.D2, params.EmbedCount, err)
			return nil
		}
	}

	compressedImg, err := jpeg.Decode(embededJpeg)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BS=%dx%d D1D2=%dx%d EC=%.2f - Decode cached error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeW, params.BlockShapeH,
			params.D1, params.D2, params.EmbedCount, err)
		return nil
	}

	// Extract
	extracted, err := watermark.Extract(ctx, compressedImg, len(params.Mark.Encoded), opts...)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BS=%dx%d D1D2=%dx%d EC=%.2f - Extract error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeW, params.BlockShapeH,
			params.D1, params.D2, params.EmbedCount, err)
		return nil
	}

	// Compare encoded
	encodedMatches := 0
	for i := range params.Mark.Encoded {
		if params.Mark.Encoded[i] == extracted[i] {
			encodedMatches++
		}
	}
	encodedAccuracy := float64(encodedMatches) / float64(len(params.Mark.Encoded)) * 100

	// Decode and compare
	decoded := params.Mark.Decode(extracted)
	decodedMatches := 0
	for i := range params.Mark.Original {
		if params.Mark.Original[i] == decoded[i] {
			decodedMatches++
		}
	}
	decodedAccuracy := float64(decodedMatches) / float64(len(params.Mark.Original)) * 100

	// Calculate SSIM
	ssim, err := calculateSSIM(params.OriginalImagePath, embeddedPath)
	if err != nil {
		log.Printf("    [WARN] Failed to calculate SSIM: %v\n", err)
	}

	duration := time.Since(start)

	var success = decodedMatches == len(params.Mark.Original)
	status := "FAIL"
	if success {
		status = "OK"
	}
	ssimStr := fmt.Sprintf(" SSIM=%.4f", ssim)
	log.Printf("    [%s] Size=%dx%d BS=%dx%d D1D2=%dx%d EC=%.2f TB=%d - E=%.1f%% D=%.1f%% T=%v%s\n",
		status, params.ImageWidth, params.ImageHeight, params.BlockShapeW, params.BlockShapeH,
		params.D1, params.D2, params.EmbedCount, params.TotalBlocks,
		encodedAccuracy, decodedAccuracy, duration, ssimStr)

	return &TestResult{&params, encodedAccuracy, decodedAccuracy, success, ssim}
}

func (params TestParams) EmbeddedImagePath(embeddedDir string) string {
	embeddedFilename := fmt.Sprintf("img%s_%dx%d_bs%dx%d_ds%dx%d.jpeg",
		params.ImageName,
		params.ImageWidth, params.ImageHeight,
		params.BlockShapeW, params.BlockShapeH,
		params.D1, params.D2)
	return filepath.Join(embeddedDir, embeddedFilename)
}

func saveEmbedImage(path string, img image.Image) (io.Reader, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100}); err != nil {
		return nil, fmt.Errorf("failed jpeg encode: %w", err)
	}
	data := buf.Bytes()

	// Save embedded image
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed create file: %w", err)
	}
	defer f.Close()

	if _, err = f.Write(data); err != nil {
		return nil, fmt.Errorf("failed write file: %w", err)
	}
	return bytes.NewReader(data), nil
}

func getEmbedImage(path string) (io.Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed open file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed read file: %w", err)
	}
	return bytes.NewReader(data), nil
}
