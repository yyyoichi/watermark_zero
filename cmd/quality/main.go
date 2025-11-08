package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "image/png"

	_ "embed"

	"github.com/yyyoichi/httpcache-go"
	watermark "github.com/yyyoichi/watermark_zero"
	innerwatermark "github.com/yyyoichi/watermark_zero/internal/watermark"
	"github.com/yyyoichi/watermark_zero/strmark/wzeromark"
	"golang.org/x/image/draw"
)

//go:embed image_urls.txt
var imageURLs []byte

// rateLimitedClient wraps an HTTP orignalClient with rate limiting between requests
// Thread-safe for concurrent requests
type rateLimitedClient struct {
	client   *http.Client
	interval time.Duration
	lastCall time.Time
	mu       sync.Mutex
}

func newRateLimitedClient(interval time.Duration) *rateLimitedClient {
	return &rateLimitedClient{
		client:   http.DefaultClient,
		interval: interval,
	}
}

func (r *rateLimitedClient) Do(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Wait if needed to maintain the interval between requests
	elapsed := time.Since(r.lastCall)
	if elapsed < r.interval {
		time.Sleep(r.interval - elapsed)
	}

	log.Println("Making request to:", req.URL.String())
	resp, err := r.client.Do(req)
	r.lastCall = time.Now()

	return resp, err
}

var orignalClient = httpcache.Client{
	Client:  newRateLimitedClient(time.Duration(250 * time.Millisecond)),
	Cache:   httpcache.NewStorageCache("/tmp/pexels_http_cache/"),
	Handler: httpcache.NewDefaultHandler(),
}

type trimClient struct {
	client httpcache.Client
}

func (r *trimClient) Do(req *http.Request) (*http.Response, error) {
	// 1 remove query parameters from the URL
	u := req.URL
	q := u.Query()
	u.RawQuery = ""
	req.URL = u
	targetWidth, err := strconv.ParseInt(q.Get("w"), 10, 64)
	if err != nil {
		return nil, err
	}
	targetHeight, err := strconv.ParseInt(q.Get("h"), 10, 64)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	src, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	srcRect := bounds
	srcRatio := float64(width) / float64(height)
	targetRatio := float64(targetWidth) / float64(targetHeight)

	if srcRatio > targetRatio {
		// ソース画像が横長すぎる場合、中央部分をトリミング
		newWidth := int(float64(height) * targetRatio)
		x := (width - newWidth) / 2
		srcRect = image.Rect(x, 0, x+newWidth, height)
	} else if srcRatio < targetRatio {
		// ソース画像が縦長すぎる場合、中央部分をトリミング
		newHeight := int(float64(width) / targetRatio)
		y := (height - newHeight) / 2
		srcRect = image.Rect(0, y, width, y+newHeight)
	}

	// リサイズ後の画像を作成
	dist := image.NewRGBA(image.Rect(0, 0, int(targetWidth), int(targetHeight)))

	// より高品質な補間方法でリサイズ
	draw.CatmullRom.Scale(dist, dist.Bounds(), src, srcRect, draw.Over, nil)

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, dist, &jpeg.Options{Quality: 100})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}
	resp.Body = io.NopCloser(&buf)
	return resp, nil
}

var client = httpcache.Client{
	Client:  &trimClient{client: orignalClient},
	Cache:   httpcache.NewStorageCache("/tmp/pexels_http_cache/"),
	Handler: httpcache.NewDefaultHandler(),
}

type TestParams struct {
	BlockShapeH int
	BlockShapeW int
	D1          int
	D2          int

	// meta
	ImageWidth  int
	EmbedCount  float64
	ImageHeight int
	TotalBlocks int
}

// Stats holds the statistics for a set of tests.
type Stats struct {
	Total         int
	Success       int
	Failures      int
	TotalAccuracy float64
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

	// Parse image URLs
	urls := parseURLs(string(imageURLs))
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

			img, err := fetchImageWithSize(url, width, height)
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

						TotalBlocks: innerwatermark.TotalBlocks(rect, innerwatermark.NewBlockShape(bs[1], bs[0])),
						ImageWidth:  width,
						ImageHeight: height,
					}
					params.EmbedCount = float64(params.TotalBlocks) / float64(len(testMark))

					// Initialize stats if not present
					d1d2Key := fmt.Sprintf("%dx%d", params.D1, params.D2)
					if _, ok := imageStats[d1d2Key]; !ok {
						imageStats[d1d2Key] = &Stats{}
					}
					if _, ok := grandTotalStats[d1d2Key]; !ok {
						grandTotalStats[d1d2Key] = &Stats{}
					}

					accuracy := testWatermark(ctx, batch, testMark, params)

					// Update stats
					imageStats[d1d2Key].Total++
					grandTotalStats[d1d2Key].Total++
					imageStats[d1d2Key].TotalAccuracy += accuracy
					grandTotalStats[d1d2Key].TotalAccuracy += accuracy

					if accuracy == 100.0 {
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
	totalAccuracy := 0.0
	log.Println("D1/D2 Pair | Avg. Accuracy | Success Rate | Success / Total")
	log.Println("-----------|---------------|--------------|-----------------")
	for d1d2, stat := range stats {
		total += stat.Total
		success += stat.Success
		totalAccuracy += stat.TotalAccuracy
		if stat.Total > 0 {
			log.Printf("%-10s | %12.2f%% | %11.2f%% | %d / %d\n",
				d1d2,
				stat.TotalAccuracy/float64(stat.Total),
				float64(stat.Success)/float64(stat.Total)*100,
				stat.Success,
				stat.Total,
			)
		}
	}
	log.Println("-----------|---------------|--------------|-----------------")
	if total > 0 {
		log.Printf("Overall    | %12.2f%% | %11.2f%% | %d / %d\n",
			totalAccuracy/float64(total),
			float64(success)/float64(total)*100,
			success,
			total,
		)
	}
	log.Println("----------------------------------------------------------")
}

func parseURLs(data string) []string {
	var urls []string
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && strings.HasPrefix(line, "http") {
			urls = append(urls, line)
		}
	}
	return urls
}

func fetchImageWithSize(url string, width, height int) (image.Image, error) {
	// Add resolution parameters
	sizeParams := fmt.Sprintf("w=%d&h=%d", width, height)
	if strings.Contains(url, "?") {
		url += "&" + sizeParams
	} else {
		url += "?" + sizeParams
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
	}

	img, err := jpeg.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode jpeg: %w", err)
	}

	return img, nil
}

func testWatermark(ctx context.Context, batch *watermark.Batch, mark []bool, params TestParams) float64 {
	opts := []watermark.Option{
		watermark.WithBlockShape(params.BlockShapeW, params.BlockShapeH),
		watermark.WithD1D2(params.D1, params.D2),
	}

	start := time.Now()

	// Embed
	markedImg, err := batch.Embed(ctx, mark, opts...)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d EmbedCount=%.2f TotalBlocks=%d - Embed error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.EmbedCount, params.TotalBlocks, err)
		return 0.0
	}

	// JPEG compression and decode with quality 100
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, markedImg, &jpeg.Options{Quality: 100}); err != nil {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d EmbedCount=%.2f TotalBlocks=%d - JPEG encode error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.EmbedCount, params.TotalBlocks, err)
		return 0.0
	}
	compressedImg, err := jpeg.Decode(&buf)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d EmbedCount=%.2f TotalBlocks=%d - JPEG decode error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.EmbedCount, params.TotalBlocks, err)
		return 0.0
	}

	// Extract
	extracted, err := watermark.Extract(ctx, compressedImg, len(mark), opts...)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d EmbedCount=%.2f TotalBlocks=%d - Extract error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.EmbedCount, params.TotalBlocks, err)
		return 0.0
	}

	// Verify
	matches := 0
	for i := range mark {
		if i < len(extracted) && mark[i] == extracted[i] {
			matches++
		}
	}

	accuracy := float64(matches) / float64(len(mark)) * 100
	duration := time.Since(start)

	if accuracy == 100.0 {
		log.Printf("    [OK] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d EmbedCount=%.2f TotalBlocks=%d - Accuracy=%.1f%% Time=%v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.EmbedCount, params.TotalBlocks, accuracy, duration)
	} else {
		log.Printf("    [FAIL] Size=%dx%d BlockShape=%dx%d D1D2=%dx%d EmbedCount=%.2f TotalBlocks=%d - Accuracy=%.1f%% Time=%v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeH, params.BlockShapeW, params.D1, params.D2, params.EmbedCount, params.TotalBlocks, accuracy, duration)
	}
	return accuracy
}
