package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"

	"exp/internal/images"
	"exp/internal/mark"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/strmark/wzeromark"
)

// OptimizeResult holds test results for visualization
type OptimizeResult struct {
	ImageSize   string
	ImageWidth  int
	ImageHeight int
	BlockShapeW int
	BlockShapeH int
	D1          int
	D2          int
	EmbedCount  float64
	TotalBlocks int

	EncodedAccuracy float64
	DecodedAccuracy float64
	Success         bool
}

const (
	TARGET_EMBED_LOW  = 1.0
	TARGET_EMBED_HIGH = 6.0
)

func main() {
	numImages := flag.Int("n", 10, "number of images to test")
	offset := flag.Int("o", 0, "offset to start from")
	flag.Parse()

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
		{36, 20},
		{30, 16},
		{25, 15},
		{24, 14},
		{23, 13},
		{22, 12},
		{21, 11},
		{20, 10},
		{19, 9},
		{18, 8},
		{17, 7},
		{15, 5},
	} // Parsse image URLs
	urls := images.ParseURLs()
	if len(urls) == 0 {
		log.Fatal("No image URLs found")
	}

	// Apply offset and limit
	if *offset >= len(urls) {
		log.Fatalf("Offset %d is beyond available images (%d)", *offset, len(urls))
	}
	urls = urls[*offset:]
	if *numImages > 0 && *numImages < len(urls) {
		urls = urls[:*numImages]
	}

	// Prepare mark (ShuffledGolay only)
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
	shuffledGolay := mark.NewShuffledGolayMark(testMark)

	log.Printf("Starting D1/D2 optimization with %d images (offset=%d)\n", len(urls), *offset)
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

					if embedCount < TARGET_EMBED_LOW || embedCount > TARGET_EMBED_HIGH {
						continue
					}

					testParams = append(testParams, TestParams{
						BlockShapeH: bs[0],
						BlockShapeW: bs[1],
						D1:          d1d2[0],
						D2:          d1d2[1],
						Mark:        shuffledGolay,
						TotalBlocks: totalBlocks,
						ImageWidth:  width,
						ImageHeight: height,
						EmbedCount:  embedCount,
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
			resultCh := make(chan TestResult, len(testParams))

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
				params := result.TestParams

				allResults = append(allResults, OptimizeResult{
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
				})
			}
		}
	}

	log.Printf("\n=== Optimization Complete ===\n")
	log.Printf("Total test results: %d\n", len(allResults))
	log.Printf("Generating visualizations...\n")

	// Generate visualizations
	outDir := "/tmp/optimize"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// 1. Scatter plot: EmbedCount vs Success Rate
	if err := generateScatterPlot(allResults, filepath.Join(outDir, "scatter_embedcount_vs_success.html")); err != nil {
		log.Printf("Failed to generate scatter plot: %v\n", err)
	} else {
		log.Printf("Generated: %s\n", filepath.Join(outDir, "scatter_embedcount_vs_success.html"))
	}

	// 2. Heatmap: D1 vs D2
	if err := generateHeatmap(allResults, filepath.Join(outDir, "heatmap_d1d2.html")); err != nil {
		log.Printf("Failed to generate heatmap: %v\n", err)
	} else {
		log.Printf("Generated: %s\n", filepath.Join(outDir, "heatmap_d1d2.html"))
	}

	log.Printf("\nAll visualizations saved to: %s\n", outDir)
}

// TestParams holds parameters for a single test
type TestParams struct {
	BlockShapeH int
	BlockShapeW int
	D1          int
	D2          int
	Mark        mark.Mark
	TotalBlocks int
	ImageWidth  int
	ImageHeight int
	EmbedCount  float64
}

// TestResult holds the test outcome
type TestResult struct {
	TestParams      *TestParams
	EncodedAccuracy float64
	DecodedAccuracy float64
	Success         bool
}

func testWatermark(ctx context.Context, batch *watermark.Batch, params TestParams) TestResult {
	opts := []watermark.Option{
		watermark.WithBlockShape(params.BlockShapeW, params.BlockShapeH),
		watermark.WithD1D2(params.D1, params.D2),
	}

	start := time.Now()

	// Embed
	markedImg, err := batch.Embed(ctx, params.Mark.Encoded, opts...)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BS=%dx%d D1D2=%dx%d EC=%.2f - Embed error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeW, params.BlockShapeH,
			params.D1, params.D2, params.EmbedCount, err)
		return TestResult{&params, 0.0, 0.0, false}
	}

	// JPEG compression
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, markedImg, &jpeg.Options{Quality: 100}); err != nil {
		log.Printf("    [FAIL] Size=%dx%d BS=%dx%d D1D2=%dx%d EC=%.2f - JPEG error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeW, params.BlockShapeH,
			params.D1, params.D2, params.EmbedCount, err)
		return TestResult{&params, 0.0, 0.0, false}
	}
	compressedImg, err := jpeg.Decode(&buf)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BS=%dx%d D1D2=%dx%d EC=%.2f - JPEG decode error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeW, params.BlockShapeH,
			params.D1, params.D2, params.EmbedCount, err)
		return TestResult{&params, 0.0, 0.0, false}
	}

	// Extract
	extracted, err := watermark.Extract(ctx, compressedImg, len(params.Mark.Encoded), opts...)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BS=%dx%d D1D2=%dx%d EC=%.2f - Extract error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeW, params.BlockShapeH,
			params.D1, params.D2, params.EmbedCount, err)
		return TestResult{&params, 0.0, 0.0, false}
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

	duration := time.Since(start)

	var success = decodedMatches == len(params.Mark.Original)
	status := "FAIL"
	if success {
		status = "OK"
	}
	log.Printf("    [%s] Size=%dx%d BS=%dx%d D1D2=%dx%d EC=%.2f TB=%d - E=%.1f%% D=%.1f%% T=%v\n",
		status, params.ImageWidth, params.ImageHeight, params.BlockShapeW, params.BlockShapeH,
		params.D1, params.D2, params.EmbedCount, params.TotalBlocks,
		encodedAccuracy, decodedAccuracy, duration)

	return TestResult{&params, encodedAccuracy, decodedAccuracy, success}
}

// generateScatterPlot creates a scatter plot of EmbedCount vs Success Rate
// Each point is colored by D1D2 and shaped by BlockShape
func generateScatterPlot(results []OptimizeResult, outputPath string) error {
	scatter := charts.NewScatter()
	scatter.SetGlobalOptions(
		// charts.WithTitleOpts(opts.Title{
		// Title: fmt.Sprintf("EmbedCount vs Success Rate (%d samples)", len(results)),
		// Subtitle: "Colored by D1D2, shaped by BlockShape",
		// }),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "EmbedCount",
			Type: "value",
			Min:  TARGET_EMBED_LOW,
			Max:  TARGET_EMBED_HIGH,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name:         "Success Rate (%)",
			NameLocation: "start",
			Type:         "value",
			Min:          60,
			Max:          100,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:     opts.Bool(true),
			Trigger:  "item",
			Position: "bottom",
		}),
	)

	resultsByD12EC := make(map[string]map[string][]OptimizeResult)
	for _, r := range results {
		d1d2Key := fmt.Sprintf("D1=%d,D2=%d", r.D1, r.D2)
		ec := fmt.Sprintf("%.1f", r.EmbedCount)
		if _, exists := resultsByD12EC[d1d2Key]; !exists {
			resultsByD12EC[d1d2Key] = make(map[string][]OptimizeResult)
		}
		resultsByD12EC[d1d2Key][ec] = append(resultsByD12EC[d1d2Key][ec], r)
	}

	// Group by D1D2 for series
	d1d2Groups := make(map[string][]opts.ScatterData)
	for d1d2Key, r := range resultsByD12EC {
		for ec, rs := range r {
			var decodedAccuracies float64
			for _, res := range rs {
				decodedAccuracies += res.DecodedAccuracy
			}
			decodedAccuracy := decodedAccuracies / float64(len(rs))
			d1d2Groups[d1d2Key] = append(d1d2Groups[d1d2Key], opts.ScatterData{
				Value:      []any{ec, decodedAccuracy},
				Symbol:     "circle",
				SymbolSize: 10,
				Name:       fmt.Sprintf("%s,EC=%s,Sample=%d", d1d2Key, ec, len(rs)),
			})
		}
	}

	// Sort keys for consistent legend order
	var d1d2Keys []string
	for k := range d1d2Groups {
		d1d2Keys = append(d1d2Keys, k)
	}
	sort.Strings(d1d2Keys)

	for _, key := range d1d2Keys {
		scatter.AddSeries(key, d1d2Groups[key])
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Print resultsByD12EC table
	log.Printf("\n=== Results by D1D2 and EmbedCount ===\n")

	// Print header
	log.Printf("%-15s | %6s | %8s | %8s | %8s\n", "D1D2", "EC", "Samples", "AvgAcc%", "Success%")
	log.Printf("%s\n", "----------------+--------+----------+----------+----------")

	for _, d1d2Key := range d1d2Keys {
		ecMap := resultsByD12EC[d1d2Key]

		// Sort EC keys in descending order
		var ecKeys []string
		for ec := range ecMap {
			ecKeys = append(ecKeys, ec)
		}
		sort.Strings(ecKeys)

		// Print each EC
		for _, ec := range ecKeys {
			rs := ecMap[ec]
			var totalAcc float64
			var successCount int
			for _, r := range rs {
				totalAcc += r.DecodedAccuracy
				if r.Success {
					successCount++
				}
			}
			avgAcc := totalAcc / float64(len(rs))
			successRate := float64(successCount) / float64(len(rs)) * 100

			log.Printf("%-15s | %6s | %8d | %7.1f%% | %7.1f%%\n",
				d1d2Key, ec, len(rs), avgAcc, successRate)
		}
	}
	log.Printf("\n")

	return scatter.Render(f)
}

// generateHeatmap creates a heatmap of D1 vs D2 with success rate as intensity
func generateHeatmap(results []OptimizeResult, outputPath string) error {
	// Aggregate success rate by D1D2
	type d1d2Key struct {
		d1, d2 int
	}
	d1d2Stats := make(map[d1d2Key]struct {
		total   int
		success int
	})

	for _, r := range results {
		key := d1d2Key{r.D1, r.D2}
		stats := d1d2Stats[key]
		stats.total++
		if r.Success {
			stats.success++
		}
		d1d2Stats[key] = stats
	}

	// Extract unique D1 and D2 values
	d1Set := make(map[int]bool)
	d2Set := make(map[int]bool)
	for key := range d1d2Stats {
		d1Set[key.d1] = true
		d2Set[key.d2] = true
	}

	var d1List, d2List []int
	for d1 := range d1Set {
		d1List = append(d1List, d1)
	}
	for d2 := range d2Set {
		d2List = append(d2List, d2)
	}
	sort.Ints(d1List)
	sort.Ints(d2List)

	// Convert D1 and D2 to string labels
	var xLabels, yLabels []string
	for _, d1 := range d1List {
		xLabels = append(xLabels, fmt.Sprintf("D1=%d", d1))
	}
	for _, d2 := range d2List {
		yLabels = append(yLabels, fmt.Sprintf("D2=%d", d2))
	}

	// Build heatmap data
	var heatmapData []opts.HeatMapData
	for i, d2 := range d2List {
		for j, d1 := range d1List {
			key := d1d2Key{d1, d2}
			stats := d1d2Stats[key]
			successRate := 0.0
			if stats.total > 0 {
				successRate = float64(stats.success) / float64(stats.total) * 100
			}
			heatmapData = append(heatmapData, opts.HeatMapData{
				Value: [3]any{j, i, successRate},
			})
		}
	}

	heatmap := charts.NewHeatMap()
	heatmap.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "D1 vs D2 Success Rate Heatmap",
			Subtitle: "Success rate (%) for each D1D2 combination",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name:      "D1",
			Type:      "category",
			Data:      xLabels,
			SplitArea: &opts.SplitArea{Show: opts.Bool(true)},
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name:      "D2",
			Type:      "category",
			Data:      yLabels,
			SplitArea: &opts.SplitArea{Show: opts.Bool(true)},
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
			Top:  "10%",
		}),
		charts.WithVisualMapOpts(opts.VisualMap{
			Calculable: opts.Bool(true),
			Min:        0,
			Max:        100,
			Range:      []float32{0, 100},
			InRange:    &opts.VisualMapInRange{Color: []string{"#313695", "#74add1", "#fee090", "#f46d43", "#a50026"}},
		}),
		charts.WithTooltipOpts(opts.Tooltip{Show: opts.Bool(true)}),
	)

	heatmap.AddSeries("Success Rate", heatmapData)

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return heatmap.Render(f)
}
