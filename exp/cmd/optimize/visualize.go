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

type DataJsonFormat struct {
	Params struct {
		ImageSizes      [][]int
		D1D2Pairs       [][]int
		BlockShapes     [][]int
		NumImages       int
		Offset          int
		TargetEmbedLow  float64
		TargetEmbedHigh float64
	}
	Results []OptimizeResult `json:"results"`
}

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

func visualizeMain(inputFile, outputDir string) {
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
	log.Printf("Parameters: NumImages=%d, Offset=%d, TargetEmbed=%.1f-%.1f\n",
		data.Params.NumImages, data.Params.Offset,
		data.Params.TargetEmbedLow, data.Params.TargetEmbedHigh)

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Generate visualizations
	log.Printf("Generating visualizations...\n")

	// Extract base name from input file (without extension)
	baseName := filepath.Base(inputFile)
	ext := filepath.Ext(baseName)
	if ext != "" {
		baseName = baseName[:len(baseName)-len(ext)]
	}

	// 1. Scatter plot: EmbedCount vs Success Rate
	scatterPath := filepath.Join(outputDir, fmt.Sprintf("scatter_embedcount_vs_success_%s.html", baseName))
	if err := generateScatterPlot(data.Results, data.Params.TargetEmbedLow, data.Params.TargetEmbedHigh, scatterPath); err != nil {
		log.Printf("Failed to generate scatter plot: %v\n", err)
	} else {
		log.Printf("Generated: %s\n", scatterPath)
	}

	// 2. Heatmap: D1 vs D2
	heatmapPath := filepath.Join(outputDir, fmt.Sprintf("heatmap_d1d2_%s.html", baseName))
	if err := generateHeatmap(data.Results, heatmapPath); err != nil {
		log.Printf("Failed to generate heatmap: %v\n", err)
	} else {
		log.Printf("Generated: %s\n", heatmapPath)
	}

	log.Printf("\nAll visualizations saved to: %s\n", outputDir)
}

// generateScatterPlot creates a scatter plot of EmbedCount vs Success Rate
// Each point is colored by D1D2 and shaped by BlockShape
func generateScatterPlot(results []OptimizeResult, targetEmbedLow, targetEmbedHigh float64, outputPath string) error {
	scatter := charts.NewScatter()
	scatter.SetGlobalOptions(
		charts.WithXAxisOpts(opts.XAxis{
			Name: "EmbedCount",
			Type: "value",
			Min:  targetEmbedLow,
			Max:  targetEmbedHigh,
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
