package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// d1d2Stats holds statistics for a D1D2 parameter combination
type d1d2Stats struct {
	d1          int
	d2          int
	avgSSIM     float64
	successRate float64
	sampleCount int
}

// qualityMain analyzes image quality degradation using SSIM
func qualityMain(inputFile, outputDir string) {
	if inputFile == "" {
		log.Fatal("Input file path is required")
	}

	// Read JSON file
	f, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Failed to open JSON file: %v", err)
	}
	defer f.Close()

	var data DataJsonFormat
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		log.Fatalf("Failed to decode JSON data: %v", err)
	}

	log.Printf("Loaded %d test results from %s\n", len(data.Results), inputFile)
	log.Printf("Calculating SSIM for each image pair...\n")

	// Group results by D1D2
	type d1d2Key struct {
		d1, d2 int
	}
	d1d2Groups := make(map[d1d2Key][]OptimizeResult)
	for _, r := range data.Results {
		key := d1d2Key{r.D1, r.D2}
		d1d2Groups[key] = append(d1d2Groups[key], r)
	}

	// Calculate SSIM and success rate for each D1D2 group
	var stats []d1d2Stats
	for key, results := range d1d2Groups {
		var totalSSIM float64
		var successCount int
		var validSSIMCount int

		for _, r := range results {
			totalSSIM += r.SSIM
			validSSIMCount++

			if r.Success {
				successCount++
			}
		}

		if validSSIMCount == 0 {
			log.Printf("Warning: No valid SSIM data for D1=%d, D2=%d\n", key.d1, key.d2)
			continue
		}

		avgSSIM := totalSSIM / float64(validSSIMCount)
		successRate := float64(successCount) / float64(len(results)) * 100

		stats = append(stats, d1d2Stats{
			d1:          key.d1,
			d2:          key.d2,
			avgSSIM:     avgSSIM,
			successRate: successRate,
			sampleCount: len(results),
		})
	}

	// Sort by D1 ascending, then D2 ascending
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].d1 == stats[j].d1 {
			return stats[i].d2 < stats[j].d2
		}
		return stats[i].d1 < stats[j].d1
	})

	log.Printf("Calculated SSIM for %d D1D2 combinations\n", len(stats))

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Extract base name from input file (without extension)
	baseName := filepath.Base(inputFile)
	ext := filepath.Ext(baseName)
	if ext != "" {
		baseName = baseName[:len(baseName)-len(ext)]
	}

	// Generate visualization
	outputPath := filepath.Join(outputDir, fmt.Sprintf("quality_ssim_vs_success_%s.html", baseName))
	if err := generateQualityChart(stats, outputPath); err != nil {
		log.Fatalf("Failed to generate quality chart: %v", err)
	}

	log.Printf("Generated: %s\n", outputPath)

	// Print summary table
	printQualityTable(stats)

	log.Printf("\nVisualization saved to: %s\n", outputDir)
}

// generateQualityChart creates a dual-axis line chart with SSIM and Success Rate
func generateQualityChart(stats []d1d2Stats, outputPath string) error {
	line := charts.NewLine()

	// Prepare X-axis data (D1*3 + D2 as numeric value)
	var xAxisData []string
	var ssimData []opts.LineData
	var successData []opts.LineData

	for _, s := range stats {
		xValue := s.d1*3 + s.d2
		xAxisData = append(xAxisData, fmt.Sprintf("%d", xValue))

		ssimData = append(ssimData, opts.LineData{
			Value: s.avgSSIM,
			Name:  fmt.Sprintf("D1=%d,D2=%d: SSIM=%.4f (n=%d)", s.d1, s.d2, s.avgSSIM, s.sampleCount),
		})
		successData = append(successData, opts.LineData{
			Value: s.successRate,
			Name:  fmt.Sprintf("D1=%d,D2=%d: Success=%.1f%% (n=%d)", s.d1, s.d2, s.successRate, s.sampleCount),
		})
	}

	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Image Quality (SSIM) vs Success Rate by D1D2",
			Subtitle: "Correlation between SSIM and watermark extraction success rate",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "D1*3 + D2",
			Type: "category",
			Data: xAxisData,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "SSIM",
			Type: "value",
			Min:  0.9,
			Max:  1.0,
			AxisLabel: &opts.AxisLabel{
				Formatter: "{value}",
			},
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
			Top:  "5%",
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "axis",
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:  "slider",
			Start: 0,
			End:   100,
		}),
	)

	// Set X-axis for line chart
	line.SetXAxis(xAxisData)

	// Add SSIM series (left Y-axis)
	line.AddSeries("SSIM", ssimData).
		SetSeriesOptions(
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			charts.WithLabelOpts(opts.Label{
				Show: opts.Bool(false),
			}),
		)

	// Extend Y-axis for dual axis (must be done before adding the second series)
	line.ExtendYAxis(opts.YAxis{
		Name: "Success Rate (%)",
		Type: "value",
		Min:  0,
		Max:  100,
		AxisLabel: &opts.AxisLabel{
			Formatter: "{value}%",
		},
	})

	// Add Success Rate series (right Y-axis)
	line.AddSeries("Success Rate (%)", successData,
		charts.WithLineChartOpts(opts.LineChart{
			Smooth:     opts.Bool(true),
			YAxisIndex: 1, // Bind to the second Y-axis (right)
		}),
		charts.WithLabelOpts(opts.Label{
			Show: opts.Bool(false),
		}),
	)

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return line.Render(f)
}

// printQualityTable prints a summary table of SSIM and success rate
func printQualityTable(stats []d1d2Stats) {
	log.Printf("\n=== Image Quality Analysis ===\n")
	log.Printf("%-15s | %8s | %10s | %8s\n", "D1D2", "SSIM", "Success%%", "Samples")
	log.Printf("%s\n", "----------------+----------+------------+----------")

	for _, s := range stats {
		log.Printf("D1=%2d,D2=%2d    | %8.6f | %9.1f%% | %8d\n",
			s.d1, s.d2, s.avgSSIM, s.successRate, s.sampleCount)
	}

	// Find optimal D1D2 (highest success rate with SSIM >= 0.99)
	var optimal *d1d2Stats
	for i := range stats {
		if stats[i].avgSSIM >= 0.99 {
			if optimal == nil || stats[i].successRate > optimal.successRate {
				optimal = &stats[i]
			}
		}
	}

	if optimal != nil {
		log.Printf("\n=== Optimal D1D2 Parameters (SSIM >= 0.99) ===\n")
		log.Printf("D1=%d, D2=%d: SSIM=%.6f, Success Rate=%.1f%% (n=%d)\n",
			optimal.d1, optimal.d2, optimal.avgSSIM, optimal.successRate, optimal.sampleCount)
	}

	log.Printf("\n")
}
