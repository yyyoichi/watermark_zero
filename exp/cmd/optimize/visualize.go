package main

import (
	"exp/internal/db"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func visualizeMain(outputDir string) {
	// Read detailed results from database using the view
	results, err := database.QueryDetailed("SELECT * FROM results_view")
	if err != nil {
		log.Fatalf("Failed to load results from database: %v", err)
	}

	log.Printf("Loaded %d test results from database\n", len(results))

	if len(results) == 0 {
		log.Fatal("No results found in database")
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Generate visualizations
	log.Printf("Generating visualizations...\n")

	// Use timestamp as base name
	baseName := "db_results"

	// 1. Scatter plot: EmbedCount vs Success Rate
	scatterPath := filepath.Join(outputDir, fmt.Sprintf("scatter_embedcount_vs_success_%s.html", baseName))
	if err := generateScatterPlot(results, scatterPath); err != nil {
		log.Printf("Failed to generate scatter plot: %v\n", err)
	} else {
		log.Printf("Generated: %s\n", scatterPath)
	}

	// 2. Heatmap: D1 vs D2
	heatmapPath := filepath.Join(outputDir, fmt.Sprintf("heatmap_d1d2_%s.html", baseName))
	if err := generateHeatmap(results, heatmapPath); err != nil {
		log.Printf("Failed to generate heatmap: %v\n", err)
	} else {
		log.Printf("Generated: %s\n", heatmapPath)
	}

	// 3. Quality chart: SSIM vs Success Rate by D1D2
	qualityPath := filepath.Join(outputDir, fmt.Sprintf("quality_ssim_vs_success_%s.html", baseName))
	if err := generateQualityChart(results, qualityPath); err != nil {
		log.Printf("Failed to generate quality chart: %v\n", err)
	} else {
		log.Printf("Generated: %s\n", qualityPath)
	}

	log.Printf("\nAll visualizations saved to: %s\n", outputDir)
}

// generateScatterPlot creates a scatter plot of EmbedCount vs Success Rate
// Each point is colored by D1D2 and shaped by BlockShape
func generateScatterPlot(results []*db.DetailedResult, outputPath string) error {
	// Calculate embed count range from results
	var minEmbed, maxEmbed float64 = 999999, 0
	for _, r := range results {
		if r.EmbedCount < minEmbed {
			minEmbed = r.EmbedCount
		}
		if r.EmbedCount > maxEmbed {
			maxEmbed = r.EmbedCount
		}
	}

	scatter := charts.NewScatter()
	scatter.SetGlobalOptions(
		charts.WithXAxisOpts(opts.XAxis{
			Name: "EmbedCount",
			Type: "value",
			Min:  minEmbed,
			Max:  maxEmbed,
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

	resultsByD12EC := make(map[string]map[string][]*db.DetailedResult)
	for _, r := range results {
		d1d2Key := fmt.Sprintf("D1=%d,D2=%d", r.D1, r.D2)
		ec := fmt.Sprintf("%.1f", r.EmbedCount)
		if _, exists := resultsByD12EC[d1d2Key]; !exists {
			resultsByD12EC[d1d2Key] = make(map[string][]*db.DetailedResult)
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
func generateHeatmap(results []*db.DetailedResult, outputPath string) error {
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

// generateQualityChart creates a chart showing SSIM distribution by BlockSize and D1D2
// X-axis: BlockSize (as categorical), grouped by D1D2 parameters
// Y-axis: SSIM values
func generateQualityChart(results []*db.DetailedResult, outputPath string) error {
	// Group results by BlockSize and D1D2
	type blockSizeKey struct {
		h, w int
	}
	type d1d2Key struct {
		d1, d2 int
	}
	type compositeKey struct {
		blockSize blockSizeKey
		d1d2      d1d2Key
	}

	// Group by BlockSize and D1D2 combination
	groups := make(map[compositeKey][]*db.DetailedResult)
	blockSizes := make(map[blockSizeKey]bool)
	d1d2Params := make(map[d1d2Key]bool)

	for _, r := range results {
		bsKey := blockSizeKey{r.BlockShapeH, r.BlockShapeW}
		d1d2K := d1d2Key{r.D1, r.D2}
		cKey := compositeKey{bsKey, d1d2K}
		groups[cKey] = append(groups[cKey], r)
		blockSizes[bsKey] = true
		d1d2Params[d1d2K] = true
	}

	// Get sorted keys
	var sortedBlockSizes []blockSizeKey
	for bs := range blockSizes {
		sortedBlockSizes = append(sortedBlockSizes, bs)
	}
	sort.Slice(sortedBlockSizes, func(i, j int) bool {
		area1 := sortedBlockSizes[i].h * sortedBlockSizes[i].w
		area2 := sortedBlockSizes[j].h * sortedBlockSizes[j].w
		return area1 < area2
	})

	var sortedD1D2 []d1d2Key
	for d := range d1d2Params {
		sortedD1D2 = append(sortedD1D2, d)
	}
	sort.Slice(sortedD1D2, func(i, j int) bool {
		if sortedD1D2[i].d1 == sortedD1D2[j].d1 {
			return sortedD1D2[i].d2 < sortedD1D2[j].d2
		}
		return sortedD1D2[i].d1 < sortedD1D2[j].d1
	})

	// Build X-axis labels: BlockSize_D1xD2 combinations
	var xLabels []string
	var labelMapping []compositeKey

	for _, bs := range sortedBlockSizes {
		for _, d1d2 := range sortedD1D2 {
			cKey := compositeKey{bs, d1d2}
			if len(groups[cKey]) > 0 {
				label := fmt.Sprintf("%dx%d\nD1=%d,D2=%d", bs.h, bs.w, d1d2.d1, d1d2.d2)
				xLabels = append(xLabels, label)
				labelMapping = append(labelMapping, cKey)
			}
		}
	}

	// Create line chart showing Median and Avg SSIM for each combination
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "SSIM Distribution by BlockSize and D1D2",
			Subtitle: "Median and Average SSIM for each BlockSize×D1D2 combination",
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "BlockSize × D1D2",
			Type: "category",
			Data: xLabels,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "SSIM",
			Type: "value",
			Min:  0.8,
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
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:   "slider",
			Orient: "vertical",
			Start:  0,
			End:    100,
		}),
	)

	// Prepare data series for Median and Avg SSIM
	var medianSSIMData []opts.LineData
	var avgSSIMData []opts.LineData

	for _, cKey := range labelMapping {
		groupResults := groups[cKey]
		var ssimValues []float64
		var totalSSIM float64

		for _, r := range groupResults {
			if r.SSIM > 0 {
				ssimValues = append(ssimValues, r.SSIM)
				totalSSIM += r.SSIM
			}
		}

		validCount := len(ssimValues)
		avgSSIM := 0.0
		medianSSIM := 0.0

		if validCount > 0 {
			avgSSIM = totalSSIM / float64(validCount)

			// Calculate median
			sort.Float64s(ssimValues)
			if validCount%2 == 0 {
				medianSSIM = (ssimValues[validCount/2-1] + ssimValues[validCount/2]) / 2
			} else {
				medianSSIM = ssimValues[validCount/2]
			}
		}

		medianSSIMData = append(medianSSIMData, opts.LineData{
			Value: medianSSIM,
			Name:  fmt.Sprintf("Median: %.4f (n=%d)", medianSSIM, validCount),
		})
		avgSSIMData = append(avgSSIMData, opts.LineData{
			Value: avgSSIM,
			Name:  fmt.Sprintf("Avg: %.4f (n=%d)", avgSSIM, validCount),
		})
	}

	line.SetXAxis(xLabels).
		AddSeries("Median SSIM", medianSSIMData,
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#ff7f0e",
				Width: 3,
			}),
			charts.WithItemStyleOpts(opts.ItemStyle{
				Color: "#ff7f0e",
			}),
		).
		AddSeries("Avg SSIM", avgSSIMData,
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: "#1f77b4",
				Width: 3,
				Type:  "dashed",
			}),
			charts.WithItemStyleOpts(opts.ItemStyle{
				Color: "#1f77b4",
			}),
		)

	// Print statistics to stdout
	fmt.Println("\n=== SSIM Distribution by BlockSize and D1D2 ===")
	fmt.Println("BlockSize\tD1D2\t\tSamples\tMedian SSIM\tAvg SSIM")
	fmt.Println("---------\t----\t\t-------\t-----------\t--------")

	for _, cKey := range labelMapping {
		groupResults := groups[cKey]
		var ssimValues []float64
		var totalSSIM float64

		for _, r := range groupResults {
			if r.SSIM > 0 {
				ssimValues = append(ssimValues, r.SSIM)
				totalSSIM += r.SSIM
			}
		}

		validCount := len(ssimValues)
		avgSSIM := 0.0
		medianSSIM := 0.0

		if validCount > 0 {
			avgSSIM = totalSSIM / float64(validCount)

			// Calculate median
			sort.Float64s(ssimValues)
			if validCount%2 == 0 {
				medianSSIM = (ssimValues[validCount/2-1] + ssimValues[validCount/2]) / 2
			} else {
				medianSSIM = ssimValues[validCount/2]
			}
		}

		fmt.Printf("%dx%d\t\tD1=%d,D2=%d\t%d\t%.4f\t\t%.4f\n",
			cKey.blockSize.h, cKey.blockSize.w, cKey.d1d2.d1, cKey.d1d2.d2,
			validCount, medianSSIM, avgSSIM)
	}
	fmt.Println()

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return line.Render(f)
}
