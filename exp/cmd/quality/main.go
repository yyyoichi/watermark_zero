package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"image/jpeg"
	"log"
	"time"

	"exp/internal/images"

	"github.com/yyyoichi/bitstream-go"
	"github.com/yyyoichi/golay"
	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/strmark/wzeromark"
)

// (image URLs are embedded and parsed inside the images package)

type TestParams struct {
	BlockShapeH int
	BlockShapeW int
	D1          int
	D2          int

	Mark Mark
	// meta
	ImageWidth  int
	EmbedCount  float64
	ImageHeight int
	TotalBlocks int
}

type Mark struct {
	Name     string
	Original []bool
	Encoded  []bool

	Decode func([]bool) []bool
}

func newGolayMark(original []bool) Mark {
	l := len(original)

	var m Mark
	m.Name = "Golay"
	m.Original = original
	{
		w := bitstream.NewBitWriter[uint64](0, 0)
		for _, v := range original {
			w.Bool(v)
		}
		data, _ := w.Data()
		var encoded []uint64
		enc := golay.NewEncoder(data, l)
		_ = enc.Encode(&encoded)
		r := bitstream.NewBitReader(encoded, 0, 0)
		r.SetBits(enc.Bits())
		m.Encoded = make([]bool, enc.Bits())
		for i := range m.Encoded {
			m.Encoded[i] = r.U8R(1, i) == 1
		}
	}
	m.Decode = func(b []bool) []bool {
		w := bitstream.NewBitWriter[uint64](0, 0)
		for _, v := range b {
			w.Bool(v)
		}
		data, _ := w.Data()
		var decoded []uint64
		dec := golay.NewDecoder(data, len(b))
		_ = dec.Decode(&decoded)
		r := bitstream.NewBitReader(decoded, 0, 0)
		r.SetBits(dec.Bits())
		result := make([]bool, dec.Bits())
		for i := range result {
			result[i] = r.U8R(1, i) == 1
		}
		return result
	}
	return m
}

// Stats holds the statistics for a set of tests.
type Stats struct {
	Total                int
	Success              int
	Failures             int
	TotalEncodedAccuracy float64 // Accuracy when comparing extracted bits with Encoded
	TotalDecodedAccuracy float64 // Accuracy when comparing decoded bits with Original
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
		{320, 180}, // 180p - EmbedCount ~1.19-2.85
		{256, 144}, // 144p - EmbedCount ~0.75-1.71
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
	golayMark := newGolayMark(testMark)

	log.Printf("Starting quality evaluation with %d images\n", len(urls))
	log.Printf("Total test cases per image: %d (image sizes) x %d (block shapes) x %d (d1/d2 pairs) = %d\n",
		len(imageSizes), len(blockShapes), len(d1d2Pairs), len(imageSizes)*len(blockShapes)*len(d1d2Pairs))

	grandTotalStats := make(map[string]*Stats)

	for i, url := range urls {
		log.Printf("\n[%d/%d] Testing image: %s\n", i+1, len(urls), url)
		imageStats := make(map[string]*Stats) // For subtotals

		for _, size := range imageSizes {
			width, height := size[0], size[1]
			log.Printf("  Size: %dx%d\n", width, height)

			img, err := images.FetchImageWithSize(url, width, height)
			if err != nil {
				log.Printf("    Error fetching image: %v\n", err)
				continue
			}

			batch := watermark.NewBatch(img)
			rect := img.Bounds()

			for _, bs := range blockShapes {
				for _, d1d2 := range d1d2Pairs {
					params := TestParams{
						BlockShapeH: bs[0],
						BlockShapeW: bs[1],
						D1:          d1d2[0],
						D2:          d1d2[1],
						Mark:        golayMark,

						TotalBlocks: (rect.Dx() + 1) / bs[1] * (rect.Dy() + 1) / bs[0],
						ImageWidth:  width,
						ImageHeight: height,
					}
					params.EmbedCount = float64(params.TotalBlocks) / float64(len(golayMark.Encoded))

					// Initialize stats if not present
					d1d2Key := fmt.Sprintf("%dx%d", params.D1, params.D2)
					if _, ok := imageStats[d1d2Key]; !ok {
						imageStats[d1d2Key] = &Stats{}
					}
					if _, ok := grandTotalStats[d1d2Key]; !ok {
						grandTotalStats[d1d2Key] = &Stats{}
					}

					result := testWatermark(ctx, batch, params)

					// Update stats
					imageStats[d1d2Key].Total++
					grandTotalStats[d1d2Key].Total++
					imageStats[d1d2Key].TotalEncodedAccuracy += result.EncodedAccuracy
					grandTotalStats[d1d2Key].TotalEncodedAccuracy += result.EncodedAccuracy
					imageStats[d1d2Key].TotalDecodedAccuracy += result.DecodedAccuracy
					grandTotalStats[d1d2Key].TotalDecodedAccuracy += result.DecodedAccuracy

					if result.EncodedAccuracy == 100.0 || result.DecodedAccuracy == 100.0 {
						imageStats[d1d2Key].Success++
						grandTotalStats[d1d2Key].Success++
					} else {
						imageStats[d1d2Key].Failures++
						grandTotalStats[d1d2Key].Failures++
					}
				}
			}
		}
		printStats(fmt.Sprintf("Subtotal for image %d (%s)", i+1, url), imageStats)
	}

	printStats("Grand Total", grandTotalStats)
}

func printStats(title string, stats map[string]*Stats) {
	log.Printf("\n--- %s ---\n", title)
	total := 0
	success := 0
	totalEncodedAccuracy := 0.0
	totalDecodedAccuracy := 0.0
	log.Println("D1/D2 Pair | Avg. Encoded Acc | Avg. Decoded Acc | Success Rate | Success / Total")
	log.Println("-----------|------------------|------------------|--------------|----------------")
	for d1d2, stat := range stats {
		total += stat.Total
		success += stat.Success
		totalEncodedAccuracy += stat.TotalEncodedAccuracy
		totalDecodedAccuracy += stat.TotalDecodedAccuracy
		if stat.Total > 0 {
			log.Printf("%-10s | %15.2f%% | %15.2f%% | %11.2f%% | %d / %d\n",
				d1d2,
				stat.TotalEncodedAccuracy/float64(stat.Total),
				stat.TotalDecodedAccuracy/float64(stat.Total),
				float64(stat.Success)/float64(stat.Total)*100,
				stat.Success,
				stat.Total,
			)
		}
	}
	log.Println("-----------|------------------|------------------|--------------|----------------")
	if total > 0 {
		log.Printf("Overall    | %15.2f%% | %15.2f%% | %11.2f%% | %d / %d\n",
			totalEncodedAccuracy/float64(total),
			totalDecodedAccuracy/float64(total),
			float64(success)/float64(total)*100,
			success,
			total,
		)
	}
	log.Println("--------------------------------------------------------------------------")
}

// TestResult holds the accuracy results for both encoded and decoded comparisons
type TestResult struct {
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
		return TestResult{0.0, 0.0}
	}

	// JPEG compression and decode with quality 100
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, markedImg, &jpeg.Options{Quality: 100}); err != nil {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d Mark=%s EmbedCount=%.2f TotalBlocks=%d - JPEG encode error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.Mark.Name, params.EmbedCount, params.TotalBlocks, err)
		return TestResult{0.0, 0.0}
	}
	compressedImg, err := jpeg.Decode(&buf)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d Mark=%s EmbedCount=%.2f TotalBlocks=%d - JPEG decode error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.Mark.Name, params.EmbedCount, params.TotalBlocks, err)
		return TestResult{0.0, 0.0}
	}

	// Extract
	extracted, err := watermark.Extract(ctx, compressedImg, len(params.Mark.Encoded), opts...)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d Mark=%s EmbedCount=%.2f TotalBlocks=%d - Extract error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.Mark.Name, params.EmbedCount, params.TotalBlocks, err)
		return TestResult{0.0, 0.0}
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
	return TestResult{encodedAccuracy, decodedAccuracy}
}
