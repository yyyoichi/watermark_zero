package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"image/jpeg"
	"log"
	"runtime"
	"time"

	"exp/internal/images"
	"exp/internal/mark"

	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/strmark/wzeromark"
)

// (image URLs are embedded and parsed inside the images package)

type TestParams struct {
	BlockShapeH int
	BlockShapeW int
	D1          int
	D2          int

	Mark mark.Mark
	// meta
	ImageWidth  int
	EmbedCount  float64
	ImageHeight int
	TotalBlocks int
}

// Stats holds the statistics for a set of tests.
type Stats struct {
	Total                int
	Success              int
	Failures             int
	TotalEncodedAccuracy float64 // Accuracy when comparing extracted bits with Encoded
	TotalDecodedAccuracy float64 // Accuracy when comparing decoded bits with Original
}

// MarkStats holds per-mark statistics for comparison
type MarkStats struct {
	EncodedAccuracy float64
	DecodedAccuracy float64
	Success         bool
}

// ParamStats holds stats for each parameter combination, organized by mark name
type ParamStats struct {
	Size       string
	BlockShape string
	D1D2       string
	MarkStats  map[string]*MarkStats // key: mark.Name
}

// ImageSizeStats holds aggregated stats per image size for an image
type ImageSizeStats struct {
	Size      string
	MarkStats map[string]*ImageSizeMarkStats // key: mark.Name
}

// ImageSizeMarkStats holds aggregated stats for a mark at a specific image size
type ImageSizeMarkStats struct {
	SuccessCount         int
	TotalTests           int
	TotalEncodedAccuracy float64
	TotalDecodedAccuracy float64
}

func boolSliceToString(b []bool) string {
	s := ""
	for _, v := range b {
		if v {
			s += "1"
		} else {
			s += "0"
		}
	}
	return s
}

func main() {
	// Parse command-line arguments
	numImages := flag.Int("n", 10, "number of images to test")
	flag.Parse()

	ctx := context.Background()

	// Test parameters: Focus on EmbedCount 5~15 range for detailed threshold analysis
	// Based on result_01.txt: threshold area is around 854x480 and below
	imageSizes := [][]int{
		{854, 480}, // 480p - EmbedCount ~9.58-17.11
		{800, 450}, // ~450p
		{768, 432}, // ~432p
		{720, 405}, // ~405p
		{640, 360}, // 360p - EmbedCount ~5.42-9.58
		{600, 338}, // ~338p
		{512, 288}, // ~288p
		{480, 270}, // ~270p
		{426, 240}, // 240p - EmbedCount ~2.39-4.28
		// {320, 180}, // 180p - EmbedCount ~1.19-2.85
		// {256, 144}, // 144p - EmbedCount ~0.75-1.71
	}

	blockShapes := [][]int{
		{6, 6},
		{6, 8},
		{8, 8},
	}

	d1d2Pairs := [][]int{
		{36, 20},
		{30, 17},
		{25, 14},
		{20, 11},
		{15, 8},
	}

	// Parse image URLs (provided by images package)
	urls := images.ParseURLs()
	if len(urls) == 0 {
		log.Fatal("No image URLs found")
	}

	// Limit the number of images to test
	if *numImages > 0 && *numImages < len(urls) {
		urls = urls[:*numImages]
	}

	seed := make([]byte, 32)
	_, _ = rand.Read(seed)
	m, err := wzeromark.New(seed, seed, "1a2b")
	if err != nil {
		log.Fatalf("Failed to create watermark: %v", err)
	}
	testMark, err := m.Encode("TEST_MARK")
	if err != nil {
		log.Fatalf("Failed to encode test mark: %v", err)
	}
	marks := []mark.Mark{
		mark.NewNormalMark(testMark),
		mark.NewGolayMark(testMark),
		mark.NewShuffledGolayMark(testMark),
	}

	log.Printf("Starting quality evaluation with %d images\n", len(urls))
	log.Printf("Total test cases per image: %d (image sizes) x %d (block shapes) x %d (d1/d2 pairs) x %d (marks) = %d\n",
		len(imageSizes), len(blockShapes), len(d1d2Pairs), len(marks), len(imageSizes)*len(blockShapes)*len(d1d2Pairs)*len(marks))

	// Grand total size-wise stats across all images
	grandTotalSizeStats := make(map[string]*ImageSizeStats)

	for i, url := range urls {
		log.Printf("\n[%d/%d] Testing image: %s\n", i+1, len(urls), url)
		imageSizeStatsMap := make(map[string]*ImageSizeStats) // for per-image size aggregation
		for _, size := range imageSizes {
			width, height := size[0], size[1]
			sizeKey := fmt.Sprintf("%dx%d", width, height)
			log.Printf("  Size: %dx%d\n", width, height)

			img, err := images.FetchImageWithSize(url, width, height)
			if err != nil {
				log.Printf("    Error fetching image: %v\n", err)
				continue
			}

			batch := watermark.NewBatch(img)
			rect := img.Bounds()

			// Initialize ImageSizeStats for this size if not exists
			if imageSizeStatsMap[sizeKey] == nil {
				imageSizeStatsMap[sizeKey] = &ImageSizeStats{
					Size:      sizeKey,
					MarkStats: make(map[string]*ImageSizeMarkStats),
				}
			}
			// Initialize grand total stats for this size if not exists
			if grandTotalSizeStats[sizeKey] == nil {
				grandTotalSizeStats[sizeKey] = &ImageSizeStats{
					Size:      sizeKey,
					MarkStats: make(map[string]*ImageSizeMarkStats),
				}
			}

			// Create test parameters channel
			numWorkers := runtime.GOMAXPROCS(0)
			testParamsCh := make(chan TestParams, numWorkers)
			resultCh := make(chan TestResult, numWorkers)

			// Count total tests for this size
			totalTestsForSize := len(blockShapes) * len(d1d2Pairs) * len(marks)

			// Start worker goroutines
			for range numWorkers {
				go func() {
					for params := range testParamsCh {
						resultCh <- testWatermark(ctx, batch, params)
					}
				}()
			}

			// Send test parameters
			go func() {
				defer close(testParamsCh)
				for _, bs := range blockShapes {
					for _, d1d2 := range d1d2Pairs {
						for _, mark := range marks {
							if imageSizeStatsMap[sizeKey].MarkStats[mark.Name] == nil {
								imageSizeStatsMap[sizeKey].MarkStats[mark.Name] = &ImageSizeMarkStats{}
							}
							if grandTotalSizeStats[sizeKey].MarkStats[mark.Name] == nil {
								grandTotalSizeStats[sizeKey].MarkStats[mark.Name] = &ImageSizeMarkStats{}
							}
							params := TestParams{
								BlockShapeH: bs[0],
								BlockShapeW: bs[1],
								D1:          d1d2[0],
								D2:          d1d2[1],
								Mark:        mark,

								TotalBlocks: ((rect.Dx() + 1) / bs[1]) * ((rect.Dy() + 1) / bs[0]),
								ImageWidth:  width,
								ImageHeight: height,
							}
							params.EmbedCount = float64(params.TotalBlocks) / float64(len(mark.Encoded))
							testParamsCh <- params
						}
					}
				}
			}()

			// Collect results
			for range totalTestsForSize {
				result := <-resultCh
				params := result.TestParams

				success := result.DecodedAccuracy == 100.0

				// Aggregate stats by image size for this image
				sizeStats := imageSizeStatsMap[sizeKey].MarkStats[params.Mark.Name]
				sizeStats.TotalTests++
				sizeStats.TotalEncodedAccuracy += result.EncodedAccuracy
				sizeStats.TotalDecodedAccuracy += result.DecodedAccuracy
				if success {
					sizeStats.SuccessCount++
				}

				// Aggregate into grand total
				grandStats := grandTotalSizeStats[sizeKey].MarkStats[params.Mark.Name]
				grandStats.TotalTests++
				grandStats.TotalEncodedAccuracy += result.EncodedAccuracy
				grandStats.TotalDecodedAccuracy += result.DecodedAccuracy
				if success {
					grandStats.SuccessCount++
				}
			}
		}

		// Print per-image size aggregated stats
		log.Printf("\n=== Image %d: Size-wise Algorithm Comparison ===\n", i+1)
		printImageSizeComparison(imageSizeStatsMap, marks, imageSizes)
	}

	// Print final overall algorithm comparison
	log.Printf("\n=== Overall Size-wise Algorithm Comparison (across all %d images) ===\n", len(urls))
	printImageSizeComparison(grandTotalSizeStats, marks, imageSizes)
}

func printImageSizeComparison(sizeStatsMap map[string]*ImageSizeStats, marks []mark.Mark, imageSizes [][]int) {
	if len(sizeStatsMap) == 0 {
		log.Println("No data for image size comparison")
		return
	}

	// Build header
	header := "Size      |"
	for _, m := range marks {
		header += fmt.Sprintf(" %-30s |", m.Name)
	}
	log.Println(header)

	separator := "----------|"
	for range marks {
		separator += "--------------------------------|"
	}
	log.Println(separator)

	// Print each size in order
	for _, size := range imageSizes {
		sizeKey := fmt.Sprintf("%dx%d", size[0], size[1])
		stats, ok := sizeStatsMap[sizeKey]
		if !ok {
			continue
		}

		row := fmt.Sprintf("%-9s |", sizeKey)
		for _, m := range marks {
			ms, ok := stats.MarkStats[m.Name]
			if !ok || ms.TotalTests == 0 {
				row += fmt.Sprintf(" %-30s |", "N/A")
				continue
			}
			avgEncoded := ms.TotalEncodedAccuracy / float64(ms.TotalTests)
			avgDecoded := ms.TotalDecodedAccuracy / float64(ms.TotalTests)
			row += fmt.Sprintf(" %d/%d (E:%.1f%% D:%.1f%%) |", ms.SuccessCount, ms.TotalTests, avgEncoded, avgDecoded)
		}
		log.Println(row)
	}
	log.Println(separator)
	log.Println("Format: Success/Total (E:AvgEncodedAcc D:AvgDecodedAcc)")
}

// TestResult holds the accuracy results for both encoded and decoded comparisons
type TestResult struct {
	*TestParams
	EncodedAccuracy float64
	DecodedAccuracy float64
}

func testWatermark(ctx context.Context, batch *watermark.Batch, params TestParams) TestResult {
	opts := []watermark.Option{
		watermark.WithBlockShape(params.BlockShapeW, params.BlockShapeH),
		watermark.WithD1D2(params.D1, params.D2),
	}

	start := time.Now()

	// Embed using Mark.Encoded
	markedImg, err := batch.Embed(ctx, params.Mark.Encoded, opts...)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d Mark=%s EmbedCount=%.2f TotalBlocks=%d - Embed error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.Mark.Name, params.EmbedCount, params.TotalBlocks, err)
		return TestResult{&params, 0.0, 0.0}
	}

	// JPEG compression and decode with quality 100
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, markedImg, &jpeg.Options{Quality: 100}); err != nil {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d Mark=%s EmbedCount=%.2f TotalBlocks=%d - JPEG encode error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.Mark.Name, params.EmbedCount, params.TotalBlocks, err)
		return TestResult{&params, 0.0, 0.0}
	}
	compressedImg, err := jpeg.Decode(&buf)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d Mark=%s EmbedCount=%.2f TotalBlocks=%d - JPEG decode error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.Mark.Name, params.EmbedCount, params.TotalBlocks, err)
		return TestResult{&params, 0.0, 0.0}
	}

	// Extract
	extracted, err := watermark.Extract(ctx, compressedImg, len(params.Mark.Encoded), opts...)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d Mark=%s EmbedCount=%.2f TotalBlocks=%d - Extract error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.Mark.Name, params.EmbedCount, params.TotalBlocks, err)
		return TestResult{&params, 0.0, 0.0}
	}

	// Verify 1: Compare extracted with Encoded
	encodedMatches := 0
	for i := range params.Mark.Encoded {
		if i < len(extracted) && params.Mark.Encoded[i] == extracted[i] {
			encodedMatches++
		}
	}
	encodedAccuracy := float64(encodedMatches) / float64(len(params.Mark.Encoded)) * 100

	// Verify 2: Decode extracted and compare with Original
	decoded := params.Mark.Decode(extracted)
	decodedMatches := 0
	for i := range params.Mark.Original {
		if i < len(decoded) && params.Mark.Original[i] == decoded[i] {
			decodedMatches++
		}
	}
	decodedAccuracy := float64(decodedMatches) / float64(len(params.Mark.Original)) * 100

	duration := time.Since(start)

	if encodedAccuracy == 100.0 || decodedAccuracy == 100.0 {
		log.Printf("    [OK] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d Mark=%s EmbedCount=%.2f TotalBlocks=%d - EncodedAcc=%.1f%% DecodedAcc=%.1f%% Time=%v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.Mark.Name, params.EmbedCount, params.TotalBlocks, encodedAccuracy, decodedAccuracy, duration)
	} else {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d Mark=%s EmbedCount=%.2f TotalBlocks=%d - EncodedAcc=%.1f%% DecodedAcc=%.1f%% Time=%v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.Mark.Name, params.EmbedCount, params.TotalBlocks, encodedAccuracy, decodedAccuracy, duration)
	}
	return TestResult{&params, encodedAccuracy, decodedAccuracy}
}
