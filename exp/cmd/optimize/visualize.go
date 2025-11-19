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

	// 3. Quality chart: SSIM vs Success Rate by D1D2
	qualityPath := filepath.Join(outputDir, fmt.Sprintf("quality_ssim_vs_success_%s.html", baseName))
	if err := generateQualityChart(data.Results, qualityPath); err != nil {
		log.Printf("Failed to generate quality chart: %v\n", err)
	} else {
		log.Printf("Generated: %s\n", qualityPath)
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
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:   "slider",
			Start:  0,
			End:    100,
			Orient: "vertical",
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:   "slider",
			Start:  0,
			End:    100,
			Orient: "horizontal",
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

// generateQualityChart creates a dual-axis line chart with SSIM and Success Rate
func generateQualityChart(results []OptimizeResult, outputPath string) error {
	// Group results by D1D2
	type d1d2Key struct {
		d1, d2 int
	}
	type d1d2Stats struct {
		d1                 int
		d2                 int
		avgSSIM            float64
		successRate        float64
		sampleCount        int
		avgDecodedAccuracy float64
		// Stats for EmbedCount < 5
		avgSSIMLow            float64
		successRateLow        float64
		sampleCountLow        int
		avgDecodedAccuracyLow float64
		// Stats for EmbedCount >= 8
		avgSSIMHigh            float64
		successRateHigh        float64
		sampleCountHigh        int
		avgDecodedAccuracyHigh float64
	}

	d1d2Groups := make(map[d1d2Key][]OptimizeResult)
	for _, r := range results {
		key := d1d2Key{r.D1, r.D2}
		d1d2Groups[key] = append(d1d2Groups[key], r)
	}

	// Calculate SSIM and success rate for each D1D2 group
	var stats []d1d2Stats
	for key, groupResults := range d1d2Groups {
		var totalSSIM float64
		var successCount int
		var validSSIMCount int
		var totalDecodedAccuracy float64

		// Stats for EmbedCount < 5
		var totalSSIMLow float64
		var successCountLow int
		var validSSIMCountLow int
		var totalDecodedAccuracyLow float64
		var countLow int

		// Stats for EmbedCount >= 8
		var totalSSIMHigh float64
		var successCountHigh int
		var validSSIMCountHigh int
		var totalDecodedAccuracyHigh float64
		var countHigh int

		for _, r := range groupResults {
			// Overall stats
			if r.SSIM > 0 {
				totalSSIM += r.SSIM
				validSSIMCount++
			}
			if r.Success {
				successCount++
			}
			totalDecodedAccuracy += r.DecodedAccuracy

			// Split by EmbedCount
			if r.EmbedCount < 5 {
				if r.SSIM > 0 {
					totalSSIMLow += r.SSIM
					validSSIMCountLow++
				}
				if r.Success {
					successCountLow++
				}
				totalDecodedAccuracyLow += r.DecodedAccuracy
				countLow++
			} else if r.EmbedCount >= 8 {
				if r.SSIM > 0 {
					totalSSIMHigh += r.SSIM
					validSSIMCountHigh++
				}
				if r.Success {
					successCountHigh++
				}
				totalDecodedAccuracyHigh += r.DecodedAccuracy
				countHigh++
			}
		}

		if validSSIMCount == 0 {
			log.Printf("Warning: No valid SSIM data for D1=%d, D2=%d\n", key.d1, key.d2)
			continue
		}

		stat := d1d2Stats{
			d1:                 key.d1,
			d2:                 key.d2,
			avgSSIM:            totalSSIM / float64(validSSIMCount),
			successRate:        float64(successCount) / float64(len(groupResults)) * 100,
			sampleCount:        len(groupResults),
			avgDecodedAccuracy: totalDecodedAccuracy / float64(len(groupResults)),
		}

		// Calculate stats for EmbedCount < 5
		if countLow > 0 && validSSIMCountLow > 0 {
			stat.avgSSIMLow = totalSSIMLow / float64(validSSIMCountLow)
			stat.successRateLow = float64(successCountLow) / float64(countLow) * 100
			stat.sampleCountLow = countLow
			stat.avgDecodedAccuracyLow = totalDecodedAccuracyLow / float64(countLow)
		}

		// Calculate stats for EmbedCount >= 8
		if countHigh > 0 && validSSIMCountHigh > 0 {
			stat.avgSSIMHigh = totalSSIMHigh / float64(validSSIMCountHigh)
			stat.successRateHigh = float64(successCountHigh) / float64(countHigh) * 100
			stat.sampleCountHigh = countHigh
			stat.avgDecodedAccuracyHigh = totalDecodedAccuracyHigh / float64(countHigh)
		}

		stats = append(stats, stat)
	}

	// Sort by D1 ascending, then D2 ascending
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].d1 == stats[j].d1 {
			return stats[i].d2 < stats[j].d2
		}
		return stats[i].d1 < stats[j].d1
	})

	line := charts.NewLine()

	// Set chart height to accommodate legend and data zoom
	line.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1400px",
			Height: "800px",
		}),
	)

	// Prepare X-axis data (D1*3 + D2 as numeric value)
	var xAxisData []string
	var ssimData []opts.LineData
	var successData []opts.LineData
	var decodedAccuracyData []opts.LineData

	// Data for EmbedCount < 5
	var ssimDataLow []opts.LineData
	var successDataLow []opts.LineData
	var decodedAccuracyDataLow []opts.LineData

	// Data for EmbedCount >= 8
	var ssimDataHigh []opts.LineData
	var successDataHigh []opts.LineData
	var decodedAccuracyDataHigh []opts.LineData

	for _, s := range stats {
		xAxisData = append(xAxisData, fmt.Sprintf("D1=%dxD2=%d", s.d1, s.d2))

		// Overall data
		ssimData = append(ssimData, opts.LineData{
			Value: s.avgSSIM,
			Name:  fmt.Sprintf("D1=%d, D2=%d: SSIM=%.4f (n=%d)", s.d1, s.d2, s.avgSSIM, s.sampleCount),
		})
		successData = append(successData, opts.LineData{
			Value: s.successRate,
			Name:  fmt.Sprintf("D1=%d, D2=%d: Success=%.1f%% (n=%d)", s.d1, s.d2, s.successRate, s.sampleCount),
		})
		decodedAccuracyData = append(decodedAccuracyData, opts.LineData{
			Value: s.avgDecodedAccuracy,
			Name:  fmt.Sprintf("D1=%d, D2=%d: DecodedAcc=%.1f%% (n=%d)", s.d1, s.d2, s.avgDecodedAccuracy, s.sampleCount),
		})

		// EmbedCount < 5 data
		if s.sampleCountLow > 0 {
			ssimDataLow = append(ssimDataLow, opts.LineData{
				Value: s.avgSSIMLow,
				Name:  fmt.Sprintf("D1=%d, D2=%d: SSIM=%.4f (n=%d, EC<5)", s.d1, s.d2, s.avgSSIMLow, s.sampleCountLow),
			})
			successDataLow = append(successDataLow, opts.LineData{
				Value: s.successRateLow,
				Name:  fmt.Sprintf("D1=%d, D2=%d: Success=%.1f%% (n=%d, EC<5)", s.d1, s.d2, s.successRateLow, s.sampleCountLow),
			})
			decodedAccuracyDataLow = append(decodedAccuracyDataLow, opts.LineData{
				Value: s.avgDecodedAccuracyLow,
				Name:  fmt.Sprintf("D1=%d, D2=%d: DecodedAcc=%.1f%% (n=%d, EC<5)", s.d1, s.d2, s.avgDecodedAccuracyLow, s.sampleCountLow),
			})
		} else {
			ssimDataLow = append(ssimDataLow, opts.LineData{Value: nil})
			successDataLow = append(successDataLow, opts.LineData{Value: nil})
			decodedAccuracyDataLow = append(decodedAccuracyDataLow, opts.LineData{Value: nil})
		}

		// EmbedCount >= 8 data
		if s.sampleCountHigh > 0 {
			ssimDataHigh = append(ssimDataHigh, opts.LineData{
				Value: s.avgSSIMHigh,
				Name:  fmt.Sprintf("D1=%d, D2=%d: SSIM=%.4f (n=%d, EC>=8)", s.d1, s.d2, s.avgSSIMHigh, s.sampleCountHigh),
			})
			successDataHigh = append(successDataHigh, opts.LineData{
				Value: s.successRateHigh,
				Name:  fmt.Sprintf("D1=%d, D2=%d: Success=%.1f%% (n=%d, EC>=8)", s.d1, s.d2, s.successRateHigh, s.sampleCountHigh),
			})
			decodedAccuracyDataHigh = append(decodedAccuracyDataHigh, opts.LineData{
				Value: s.avgDecodedAccuracyHigh,
				Name:  fmt.Sprintf("D1=%d, D2=%d: DecodedAcc=%.1f%% (n=%d, EC>=8)", s.d1, s.d2, s.avgDecodedAccuracyHigh, s.sampleCountHigh),
			})
		} else {
			ssimDataHigh = append(ssimDataHigh, opts.LineData{Value: nil})
			successDataHigh = append(successDataHigh, opts.LineData{Value: nil})
			decodedAccuracyDataHigh = append(decodedAccuracyDataHigh, opts.LineData{Value: nil})
		}
	}

	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Image Quality (SSIM) vs Success Rate by D1D2",
			Subtitle: "Correlation between SSIM and watermark extraction success rate",
			Top:      "2%",
			Left:     "center",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "D1*3 + D2",
			Type: "category",
			Data: xAxisData,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "SSIM",
			Type: "value",
			Min:  0.97,
			Max:  1.0,
			AxisLabel: &opts.AxisLabel{
				Formatter: "{value}",
			},
		}),
		charts.WithLegendOpts(opts.Legend{
			Show:   opts.Bool(true),
			Top:    "12%",
			Left:   "center",
			Orient: "horizontal",
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
		charts.WithGridOpts(opts.Grid{
			Top:    "25%",
			Bottom: "15%",
			Left:   "8%",
			Right:  "8%",
		}),
	)

	// Set X-axis for line chart
	line.SetXAxis(xAxisData)

	// Add SSIM series (left Y-axis) - Blue color family
	line.AddSeries("SSIM (All)", ssimData,
		charts.WithLineChartOpts(opts.LineChart{
			Smooth: opts.Bool(true),
		}),
		charts.WithLabelOpts(opts.Label{
			Show: opts.Bool(false),
		}),
		charts.WithItemStyleOpts(opts.ItemStyle{
			Color: "#1f77b4", // Solid blue
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Color: "#1f77b4",
			Width: 3,
			Type:  "solid",
		}),
	)

	line.AddSeries("SSIM (EC<5)", ssimDataLow,
		charts.WithLineChartOpts(opts.LineChart{
			Smooth: opts.Bool(true),
		}),
		charts.WithLabelOpts(opts.Label{
			Show: opts.Bool(false),
		}),
		charts.WithItemStyleOpts(opts.ItemStyle{
			Color: "#1f77b4", // Same blue
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Color: "#1f77b4",
			Width: 2,
			Type:  "dashed",
		}),
	)

	line.AddSeries("SSIM (EC>=8)", ssimDataHigh,
		charts.WithLineChartOpts(opts.LineChart{
			Smooth: opts.Bool(true),
		}),
		charts.WithLabelOpts(opts.Label{
			Show: opts.Bool(false),
		}),
		charts.WithItemStyleOpts(opts.ItemStyle{
			Color: "#1f77b4", // Same blue
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Color: "#1f77b4",
			Width: 2,
			Type:  "dotted",
		}),
	)

	// Extend Y-axis for dual axis (must be done before adding the second series)
	line.ExtendYAxis(opts.YAxis{
		Name: "Success Rate (%)",
		Type: "value",
		Min:  40,
		Max:  100,
		AxisLabel: &opts.AxisLabel{
			Formatter: "{value}%",
		},
	})

	// Add Success Rate series (right Y-axis) - Green color family
	line.AddSeries("Success Rate (All)", successData,
		charts.WithLineChartOpts(opts.LineChart{
			Smooth:     opts.Bool(true),
			YAxisIndex: 1, // Bind to the second Y-axis (right)
		}),
		charts.WithLabelOpts(opts.Label{
			Show: opts.Bool(false),
		}),
		charts.WithItemStyleOpts(opts.ItemStyle{
			Color: "#2ca02c", // Solid green
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Color: "#2ca02c",
			Width: 3,
			Type:  "solid",
		}),
	)

	line.AddSeries("Success Rate (EC<5)", successDataLow,
		charts.WithLineChartOpts(opts.LineChart{
			Smooth:     opts.Bool(true),
			YAxisIndex: 1,
		}),
		charts.WithLabelOpts(opts.Label{
			Show: opts.Bool(false),
		}),
		charts.WithItemStyleOpts(opts.ItemStyle{
			Color: "#2ca02c", // Same green
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Color: "#2ca02c",
			Width: 2,
			Type:  "dashed",
		}),
	)

	line.AddSeries("Success Rate (EC>=8)", successDataHigh,
		charts.WithLineChartOpts(opts.LineChart{
			Smooth:     opts.Bool(true),
			YAxisIndex: 1,
		}),
		charts.WithLabelOpts(opts.Label{
			Show: opts.Bool(false),
		}),
		charts.WithItemStyleOpts(opts.ItemStyle{
			Color: "#2ca02c", // Same green
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Color: "#2ca02c",
			Width: 2,
			Type:  "dotted",
		}),
	)

	// Add Decoded Accuracy series (right Y-axis) - Orange color family
	line.AddSeries("Avg Decoded Accuracy (All)", decodedAccuracyData,
		charts.WithLineChartOpts(opts.LineChart{
			Smooth:     opts.Bool(true),
			YAxisIndex: 1, // Bind to the second Y-axis (right)
		}),
		charts.WithLabelOpts(opts.Label{
			Show: opts.Bool(false),
		}),
		charts.WithItemStyleOpts(opts.ItemStyle{
			Color: "#ff7f0e", // Solid orange
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Color: "#ff7f0e",
			Width: 3,
			Type:  "solid",
		}),
	)

	line.AddSeries("Avg Decoded Accuracy (EC<5)", decodedAccuracyDataLow,
		charts.WithLineChartOpts(opts.LineChart{
			Smooth:     opts.Bool(true),
			YAxisIndex: 1,
		}),
		charts.WithLabelOpts(opts.Label{
			Show: opts.Bool(false),
		}),
		charts.WithItemStyleOpts(opts.ItemStyle{
			Color: "#ff7f0e", // Same orange
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Color: "#ff7f0e",
			Width: 2,
			Type:  "dashed",
		}),
	)

	line.AddSeries("Avg Decoded Accuracy (EC>=8)", decodedAccuracyDataHigh,
		charts.WithLineChartOpts(opts.LineChart{
			Smooth:     opts.Bool(true),
			YAxisIndex: 1,
		}),
		charts.WithLabelOpts(opts.Label{
			Show: opts.Bool(false),
		}),
		charts.WithItemStyleOpts(opts.ItemStyle{
			Color: "#ff7f0e", // Same orange
		}),
		charts.WithLineStyleOpts(opts.LineStyle{
			Color: "#ff7f0e",
			Width: 2,
			Type:  "dotted",
		}),
	)

	// Print statistics to stdout
	fmt.Println("\n=== Quality Chart Data ===")
	fmt.Println("D1D2\t\tSSIM(All)\tSSIM(EC<5)\tSSIM(EC>=8)\tSuccess(All)\tSuccess(EC<5)\tSuccess(EC>=8)")
	fmt.Println("----\t\t---------\t----------\t-----------\t------------\t-------------\t--------------")
	for _, s := range stats {
		fmt.Printf("D1=%dx%d\t%.4f\t\t", s.d1, s.d2, s.avgSSIM)
		if s.sampleCountLow > 0 {
			fmt.Printf("%.4f\t\t", s.avgSSIMLow)
		} else {
			fmt.Printf("N/A\t\t")
		}
		if s.sampleCountHigh > 0 {
			fmt.Printf("%.4f\t\t", s.avgSSIMHigh)
		} else {
			fmt.Printf("N/A\t\t")
		}
		fmt.Printf("%.1f%%\t\t", s.successRate)
		if s.sampleCountLow > 0 {
			fmt.Printf("%.1f%%\t\t", s.successRateLow)
		} else {
			fmt.Printf("N/A\t\t")
		}
		if s.sampleCountHigh > 0 {
			fmt.Printf("%.1f%%\n", s.successRateHigh)
		} else {
			fmt.Printf("N/A\n")
		}
	}
	fmt.Println()

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return line.Render(f)
}
