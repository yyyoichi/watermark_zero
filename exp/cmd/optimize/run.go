package main

import (
	"bytes"
	"context"
	"exp/internal/db"
	"exp/internal/images"
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

	watermark "github.com/yyyoichi/watermark_zero"
	"github.com/yyyoichi/watermark_zero/mark"
)

func runMain(numImages, offset int) {
	ctx := context.Background()

	// Parse image URLs
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

	log.Printf("Starting D1/D2 optimization with %d images (offset=%d)\n", len(urls), offset)

	dbMarks, err := database.ListMarks()
	if err != nil {
		log.Fatalf("Failed to list marks: %v", err)
	}
	if len(dbMarks) == 0 {
		log.Fatal("No marks found in database")
	}
	dbMark := dbMarks[0] // Use the first mark for testing
	algos, err := database.ListMarkEccAlgos()
	if err != nil {
		log.Fatalf("Failed to list mark ECC algos: %v", err)
	}
	var marks = make([]TestMark, 0, len(algos))
	{
		for _, algo := range algos {
			switch algo.AlgoName {
			case EccAlgoShuffledGolay:
				m := TestMark{
					algo:     algo,
					original: dbMark,
					encoded:  mark.NewBytes(dbMark.Mark),
				}
				marks = append(marks, m)

			case EccAlgoNoEcc:
				m := TestMark{
					algo:     algo,
					original: dbMark,
					encoded:  mark.NewBytes(dbMark.Mark, mark.WithoutECC()),
				}
				marks = append(marks, m)
			}
		}
	}
	// Get all image sizes for this image from DB
	imageSizes, err := database.ListImageSizes()
	if err != nil {
		log.Printf("Failed to get image sizes: %v\n", err)
	}
	// Get mark params
	markParams, err := database.ListMarkParams()
	if err != nil {
		log.Printf("Failed to list mark params: %v", err)
	}

	for i, url := range urls {
		log.Printf("\n[%d/%d] Testing image: %s\n", i+1, len(urls), url)

		// Get image ID from map (already registered in init)
		imageID, err := database.InsertImage(url)
		if err != nil {
			log.Printf("Failed to insert image %s: %v", url, err)
			continue
		}

		for _, imageSize := range imageSizes {
			width, height := imageSize.Width, imageSize.Height
			sizeKey := fmt.Sprintf("%dx%d", width, height)
			log.Printf("  Size: %s\n", sizeKey)

			img, err := images.FetchImageWithSize(url, width, height)
			if err != nil {
				log.Printf("    Error fetching image: %v\n", err)
				continue
			}

			batch := watermark.NewBatch(img)
			rect := img.Bounds()

			var testParams []TestParams
			for _, markParam := range markParams {
				totalBlocks := ((rect.Dx() + 1) / markParam.BlockShapeW) * ((rect.Dy() + 1) / markParam.BlockShapeH)
				for _, mk := range marks {
					embedCount := float64(totalBlocks) / float64(mk.encoded.Len())
					if embedCount < 1.0 || embedCount >= 16.0 {
						continue
					}

					if resultID, err := database.ResultExists(imageID, imageSize.ID, mk.original.ID, mk.algo.ID, markParam.ID); err != nil {
						log.Printf("    Failed to check existing result: %v", err)
						continue
					} else if resultID != 0 {
						// continue
					}
					testParams = append(testParams, TestParams{
						ImageID:     imageID,
						ImageSizeID: imageSize.ID,
						MarkID:      mk.original.ID,
						EccAlgoID:   mk.algo.ID,
						MarkParamID: markParam.ID,

						BlockShapeW: markParam.BlockShapeW,
						BlockShapeH: markParam.BlockShapeH,
						D1:          markParam.D1,
						D2:          markParam.D2,
						ImageWidth:  width,
						ImageHeight: height,

						Mark: mk,

						TotalBlocks:       totalBlocks,
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
				// Insert result to database
				dbResult := &db.Result{
					ImageID:         params.ImageID,
					ImageSizeID:     params.ImageSizeID,
					MarkID:          params.MarkID,
					MarkEccAlgoID:   params.Mark.algo.ID,
					MarkParamID:     params.MarkParamID,
					EmbedCount:      params.EmbedCount,
					TotalBlocks:     params.TotalBlocks,
					EncodedAccuracy: result.EncodedAccuracy,
					DecodedAccuracy: result.DecodedAccuracy,
					Success:         result.Success,
					SSIM:            result.SSIM,
				}

				if _, err := database.InsertResult(dbResult); err != nil {
					log.Printf("Failed to insert result: %v", err)
				}
			}
		}
	}
}

// TestParams holds parameters for a single test
type TestParams struct {
	ImageID     int64
	ImageSizeID int64
	MarkID      int64
	EccAlgoID   int64
	MarkParamID int64

	BlockShapeW, BlockShapeH int
	D1, D2                   int
	ImageWidth, ImageHeight  int

	Mark              TestMark
	TotalBlocks       int
	EmbedCount        float64
	ImageName         string
	OriginalImagePath string
}
type TestMark struct {
	algo     *db.MarkEccAlgo
	original *db.Mark
	encoded  *mark.Mark64
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
		embeddedImg, err := batch.Embed(ctx, params.Mark.encoded, opts...)
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
	extracted, err := watermark.Extract(ctx, compressedImg, params.Mark.encoded, opts...)
	if err != nil {
		log.Printf("    [FAIL] Size=%dx%d BS=%dx%d D1D2=%dx%d EC=%.2f - Extract error: %v\n",
			params.ImageWidth, params.ImageHeight, params.BlockShapeW, params.BlockShapeH,
			params.D1, params.D2, params.EmbedCount, err)
		return nil
	}

	encodedAccuracy, decodedAccuracy, success := calcAccuracy(params.Mark.encoded, extracted.(*mark.Mark64))

	// Calculate SSIM
	ssim, err := calculateSSIM(params.OriginalImagePath, embeddedPath)
	if err != nil {
		log.Printf("    [WARN] Failed to calculate SSIM: %v\n", err)
	}

	duration := time.Since(start)

	status := "FAIL"
	if success {
		status = "OK"
	}
	ssimStr := fmt.Sprintf(" SSIM=%.4f", ssim)
	log.Printf("    [%s] Size=%dx%d BS=%dx%d D1D2=%dx%d EC=%.2f TB=%d Algo=%s - E=%.1f%% D=%.1f%% T=%v%s\n",
		status, params.ImageWidth, params.ImageHeight, params.BlockShapeW, params.BlockShapeH,
		params.D1, params.D2, params.EmbedCount, params.TotalBlocks, params.Mark.algo.AlgoName,
		encodedAccuracy, decodedAccuracy, duration, ssimStr)

	return &TestResult{&params, encodedAccuracy, decodedAccuracy, success, ssim}
}

func (params TestParams) EmbeddedImagePath(embeddedDir string) string {
	embeddedFilename := fmt.Sprintf("img%s_%dx%d_bs%dx%d_ds%dx%d_%s.jpeg",
		params.ImageName,
		params.ImageWidth, params.ImageHeight,
		params.BlockShapeW, params.BlockShapeH,
		params.D1, params.D2,
		params.Mark.algo.AlgoName,
	)
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

func calcAccuracy(want, got *mark.Mark64) (encodedAccuracy, decodedAccuracy float64, success bool) {
	encodedMatches := 0
	for i := range want.Len() {
		if want.GetBit(i) == got.GetBit(i) {
			encodedMatches++
		}
	}
	encodedAccuracy = float64(encodedMatches) / float64(want.Len()) * 100

	decodedWant := want.DecodeToBools()
	decodedGot := got.DecodeToBools()
	decodedMatches := 0
	for i := range decodedWant {
		if decodedWant[i] == decodedGot[i] {
			decodedMatches++
		}
	}
	decodedAccuracy = float64(decodedMatches) / float64(len(decodedWant)) * 100
	success = decodedMatches == len(decodedWant)
	return
}
