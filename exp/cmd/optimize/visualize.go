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
	// Exclude PNG files and filter to EmbedCount < 16
	results, err := database.QueryDetailed("SELECT * FROM results_view WHERE image_uri NOT LIKE '%.png' AND embed_count < 16")
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

	// 1. SSIM comparison by parameters (BlockSize, D1D2)
	ssimPath := filepath.Join(outputDir, fmt.Sprintf("ssim_by_params_%s.html", baseName))
	if err := generateSSIMByParamsChart(results, ssimPath); err != nil {
		log.Printf("Failed to generate SSIM comparison chart: %v\n", err)
	} else {
		log.Printf("Generated: %s\n", ssimPath)
	}

	// 2. Success rate comparison by parameters (D1D2, EmbedCount thresholds, algorithms)
	chartPath := filepath.Join(outputDir, fmt.Sprintf("success_rate_by_params_%s.html", baseName))
	if err := generateSuccessRateByParamsChart(results, chartPath, "Success Rate Comparison by Parameters"); err != nil {
		log.Printf("Failed to generate success rate comparison chart: %v\n", err)
	} else {
		log.Printf("Generated: %s\n", chartPath)
	}

	// 2-1. Success rate comparison by parameters (S-Golay, 8x8 block only)
	sgolayChartPath := filepath.Join(outputDir, fmt.Sprintf("success_rate_by_params_sgolay_8x8_%s.html", baseName))
	sgolayQuery := fmt.Sprintf(
		"SELECT * FROM results_view WHERE image_uri NOT LIKE '%%.png' AND embed_count < 16 AND ecc_algo = '%s' AND block_shape_h = 8 AND block_shape_w = 8",
		EccAlgoShuffledGolay,
	)
	sgolayResults, err := database.QueryDetailed(sgolayQuery)
	if err != nil {
		log.Printf("Failed to load filtered S-Golay 8x8 results: %v\n", err)
	} else {
		if err := generateSuccessRateByParamsChart(sgolayResults, sgolayChartPath, "Success Rate Comparison (S-Golay, 8×8 Block)"); err != nil {
			log.Printf("Failed to generate S-Golay 8x8 success rate comparison chart: %v\n", err)
		} else {
			log.Printf("Generated: %s\n", sgolayChartPath)
		}
	}

	// 3. Combined Success Rate & SSIM by EmbedCount (S-Golay, 8x8, D1=21, 7<=D2<=11)
	combinedECPath := filepath.Join(outputDir, fmt.Sprintf("combined_successrate_ssim_by_embedcount_%s.html", baseName))
	combinedECQuery := fmt.Sprintf(
		"SELECT * FROM results_view WHERE image_uri NOT LIKE '%%.png' AND embed_count < 31 AND ecc_algo = '%s' AND block_shape_h = 8 AND block_shape_w = 8 AND d1 = 21 AND d2 >= 7 AND d2 <= 11",
		EccAlgoShuffledGolay,
	)
	combinedECResults, err := database.QueryDetailed(combinedECQuery)
	if err != nil {
		log.Printf("Failed to load filtered results for combined EC chart: %v\n", err)
	} else {
		if err := generateCombinedSuccessSSIMByEmbedCountChart(combinedECResults, combinedECPath, "Success Rate & SSIM by EmbedCount (S-Golay, 8×8, D1=21, 7≤D2≤11)"); err != nil {
			log.Printf("Failed to generate combined success rate & SSIM by EmbedCount chart: %v\n", err)
		} else {
			log.Printf("Generated: %s\n", combinedECPath)
		}
	}

	log.Printf("\nAll visualizations saved to: %s\n", outputDir)
}

// generateSuccessRateByParamsChart creates a line chart comparing success rates across parameters
// X-axis: D1D2 combinations
// Y-axis: Success Rate (%)
// Lines: Different algorithms with EmbedCount thresholds (>=1, >=4, >=8, >=10, >=12, >=14, >=15)
// title: chart title
func generateSuccessRateByParamsChart(results []*db.DetailedResult, outputPath string, title string) error {
	type d1d2Key struct {
		d1, d2 int
	}

	// EmbedCount thresholds to analyze
	thresholds := []float64{1, 4, 8, 10, 12, 14, 15}

	// Group results by algorithm, D1D2, and EmbedCount
	// Map: algo -> d1d2 -> embedCount -> results
	groupedResults := make(map[string]map[d1d2Key]map[float64][]*db.DetailedResult)
	d1d2Set := make(map[d1d2Key]bool)
	algoSet := make(map[string]bool)

	for _, r := range results {
		algoSet[r.ECCAlgo] = true
		if groupedResults[r.ECCAlgo] == nil {
			groupedResults[r.ECCAlgo] = make(map[d1d2Key]map[float64][]*db.DetailedResult)
		}
		key := d1d2Key{r.D1, r.D2}
		d1d2Set[key] = true

		if groupedResults[r.ECCAlgo][key] == nil {
			groupedResults[r.ECCAlgo][key] = make(map[float64][]*db.DetailedResult)
		}
		groupedResults[r.ECCAlgo][key][r.EmbedCount] = append(groupedResults[r.ECCAlgo][key][r.EmbedCount], r)
	}

	// Sort D1D2 keys
	var sortedD1D2 []d1d2Key
	for k := range d1d2Set {
		sortedD1D2 = append(sortedD1D2, k)
	}
	sort.Slice(sortedD1D2, func(i, j int) bool {
		if sortedD1D2[i].d1 != sortedD1D2[j].d1 {
			return sortedD1D2[i].d1 < sortedD1D2[j].d1
		}
		return sortedD1D2[i].d2 < sortedD1D2[j].d2
	})

	// Sort algorithms
	var sortedAlgos []string
	for algo := range algoSet {
		sortedAlgos = append(sortedAlgos, algo)
	}
	sort.Strings(sortedAlgos)

	// Build X-axis labels: show D1×D2 format
	var xLabels []string
	for _, key := range sortedD1D2 {
		xLabels = append(xLabels, fmt.Sprintf("%d×%d", key.d1, key.d2))
	}

	// Create line chart
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: title,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "D1 × D2",
			Type: "category",
			Data: xLabels,
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Success Rate (%)",
			Type: "value",
			Min:  0,
			Max:  100,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "axis",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
			Top:  "5%",
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

	// Color palette for different algorithms
	algoColors := map[string]string{
		"RS":     "#1f77b4", // blue
		"BCH":    "#ff7f0e", // orange
		"LDPC":   "#2ca02c", // green
		"Turbo":  "#d62728", // red
		"Polar":  "#9467bd", // purple
		"Repeat": "#8c564b", // brown
	}

	// Line styles for different thresholds
	thresholdStyles := map[float64]string{
		1:  "solid",
		4:  "dashed",
		7:  "dotted",
		8:  "solid",
		9:  "dashed",
		10: "dotted",
	}

	// Print statistics header
	fmt.Printf("\n=== %s ===\n", title)
	fmt.Println("Algorithm\tThreshold\tD1D2\t\tSamples\tSuccess%")
	fmt.Println("---------\t---------\t----\t\t-------\t--------")

	// Set X-axis with labels
	line.SetXAxis(xLabels)

	// Add series for each algorithm-threshold combination
	for _, algo := range sortedAlgos {
		for _, threshold := range thresholds {
			var lineData []opts.LineData

			for _, d1d2 := range sortedD1D2 {
				// Calculate average success rate for EmbedCount >= threshold
				var totalSuccess, totalCount int

				if ecMap, exists := groupedResults[algo][d1d2]; exists {
					for ec, rs := range ecMap {
						if ec >= threshold {
							for _, r := range rs {
								totalCount++
								if r.Success {
									totalSuccess++
								}
							}
						}
					}
				}

				successRate := 0.0
				if totalCount > 0 {
					successRate = float64(totalSuccess) / float64(totalCount) * 100
				}

				lineData = append(lineData, opts.LineData{
					Value: successRate,
					Name:  fmt.Sprintf("%s EC>=%.0f D1=%d,D2=%d (n=%d)", algo, threshold, d1d2.d1, d1d2.d2, totalCount),
				})

				// Print statistics
				if totalCount > 0 {
					fmt.Printf("%s\t\t>=%.0f\t\tD1=%d,D2=%d\t%d\t%.1f%%\n",
						algo, threshold, d1d2.d1, d1d2.d2, totalCount, successRate)
				}
			}

			// Get color for algorithm
			color, ok := algoColors[algo]
			if !ok {
				color = "#808080" // gray for unknown
			}

			// Get line style for threshold
			lineStyle, ok := thresholdStyles[threshold]
			if !ok {
				lineStyle = "solid"
			}

			// Add series
			seriesName := fmt.Sprintf("%s (EC>=%.0f)", algo, threshold)
			line.AddSeries(seriesName, lineData,
				charts.WithLineChartOpts(opts.LineChart{
					Smooth: opts.Bool(true),
				}),
				charts.WithLineStyleOpts(opts.LineStyle{
					Color: color,
					Width: 2,
					Type:  lineStyle,
				}),
				charts.WithItemStyleOpts(opts.ItemStyle{
					Color: color,
				}),
			)
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

// generateD1D2SuccessRateHeatmap creates a heatmap showing success rate for each D1×D2 combination
// Filters to only include S-Golay algorithm results
// (removed) generateD1D2SuccessRateHeatmap was deleted per request

// generateCombinedSuccessSSIMByEmbedCountChart creates a dual Y-axis line chart
// X-axis: EmbedCount (integer, floored)
// Left Y-axis: Success Rate (%)
// Right Y-axis: SSIM
// Lines: Different D1D2 parameter combinations
// Each line shows the average success rate and SSIM for that specific EmbedCount value (not cumulative)
func generateCombinedSuccessSSIMByEmbedCountChart(results []*db.DetailedResult, outputPath string, title string) error {
	type d1d2Key struct {
		d1, d2 int
	}

	// Group results by D1D2 and EmbedCount (floored)
	// Map: d1d2 -> embedCount (int) -> results
	groupedResults := make(map[d1d2Key]map[int][]*db.DetailedResult)
	d1d2Set := make(map[d1d2Key]bool)
	embedCountSet := make(map[int]bool)

	for _, r := range results {
		key := d1d2Key{r.D1, r.D2}
		d1d2Set[key] = true
		ec := int(r.EmbedCount) // floor to int
		embedCountSet[ec] = true

		if groupedResults[key] == nil {
			groupedResults[key] = make(map[int][]*db.DetailedResult)
		}
		groupedResults[key][ec] = append(groupedResults[key][ec], r)
	}

	// Sort D1D2 keys
	var sortedD1D2 []d1d2Key
	for k := range d1d2Set {
		sortedD1D2 = append(sortedD1D2, k)
	}
	sort.Slice(sortedD1D2, func(i, j int) bool {
		if sortedD1D2[i].d1 != sortedD1D2[j].d1 {
			return sortedD1D2[i].d1 < sortedD1D2[j].d1
		}
		return sortedD1D2[i].d2 < sortedD1D2[j].d2
	})

	// Sort EmbedCount keys
	var sortedEC []int
	for ec := range embedCountSet {
		sortedEC = append(sortedEC, ec)
	}
	sort.Ints(sortedEC)

	// Build X-axis labels: EmbedCount values
	var xLabels []string
	for _, ec := range sortedEC {
		xLabels = append(xLabels, fmt.Sprintf("%d", ec))
	}

	// Create line chart
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: title,
		}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "EmbedCount",
			Type: "category",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name: "Success Rate (%)",
			Type: "value",
			Min:  0,
			Max:  100,
		}),
		charts.WithTooltipOpts(opts.Tooltip{
			Show:    opts.Bool(true),
			Trigger: "axis",
		}),
		charts.WithLegendOpts(opts.Legend{
			Show: opts.Bool(true),
			Top:  "5%",
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

	// Extend YAxis for dual axis (SSIM on the right)
	line.ExtendYAxis(opts.YAxis{
		Name: "SSIM",
		Type: "value",
		Min:  0.8,
		Max:  1.0,
	})

	// Color palette for different D1D2 combinations
	colors := []string{"#1f77b4", "#ff7f0e", "#2ca02c", "#d62728", "#9467bd", "#8c564b", "#e377c2", "#7f7f7f", "#bcbd22", "#17becf"}

	// Print statistics header
	fmt.Printf("\n=== %s ===\n", title)
	fmt.Println("D1D2\t\tEC\tSamples\tSuccess%%\tSSIM")
	fmt.Println("----\t\t--\t-------\t--------\t----")

	// Set X-axis with labels
	line.SetXAxis(xLabels)

	// Add series for each D1D2 combination
	for idx, d1d2 := range sortedD1D2 {
		var successData []opts.LineData
		var ssimData []opts.LineData

		for _, ec := range sortedEC {
			// Calculate average success rate and SSIM for this specific EmbedCount
			var totalSuccess, totalCount int
			var totalSSIM float64
			var ssimCount int

			if ecResults, exists := groupedResults[d1d2][ec]; exists {
				for _, r := range ecResults {
					totalCount++
					if r.Success {
						totalSuccess++
					}
					if r.SSIM > 0 {
						totalSSIM += r.SSIM
						ssimCount++
					}
				}
			}

			successRate := 0.0
			if totalCount > 0 {
				successRate = float64(totalSuccess) / float64(totalCount) * 100
			}

			avgSSIM := 0.0
			if ssimCount > 0 {
				avgSSIM = totalSSIM / float64(ssimCount)
			}

			successData = append(successData, opts.LineData{
				Value: successRate,
				Name:  fmt.Sprintf("D1=%d,D2=%d EC=%d Success: %.1f%% (n=%d)", d1d2.d1, d1d2.d2, ec, successRate, totalCount),
			})

			ssimData = append(ssimData, opts.LineData{
				Value: avgSSIM,
				Name:  fmt.Sprintf("D1=%d,D2=%d EC=%d SSIM: %.4f (n=%d)", d1d2.d1, d1d2.d2, ec, avgSSIM, ssimCount),
			})

			// Print statistics
			if totalCount > 0 {
				fmt.Printf("D1=%d,D2=%d\t%d\t%d\t%.1f%%\t\t%.4f\n",
					d1d2.d1, d1d2.d2, ec, totalCount, successRate, avgSSIM)
			}
		}

		// Get color for this D1D2
		color := colors[idx%len(colors)]

		// Add success rate series (left Y-axis)
		seriesName := fmt.Sprintf("Success Rate (D1=%d,D2=%d)", d1d2.d1, d1d2.d2)
		line.AddSeries(seriesName, successData,
			charts.WithLineChartOpts(opts.LineChart{
				Smooth:     opts.Bool(true),
				YAxisIndex: 0,
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: color,
				Width: 2,
			}),
			charts.WithItemStyleOpts(opts.ItemStyle{
				Color: color,
			}),
		)

		// Add SSIM series (right Y-axis) with dashed line
		ssimSeriesName := fmt.Sprintf("SSIM (D1=%d,D2=%d)", d1d2.d1, d1d2.d2)
		line.AddSeries(ssimSeriesName, ssimData,
			charts.WithLineChartOpts(opts.LineChart{
				Smooth:     opts.Bool(true),
				YAxisIndex: 1,
			}),
			charts.WithLineStyleOpts(opts.LineStyle{
				Color: color,
				Width: 2,
				Type:  "dashed",
			}),
			charts.WithItemStyleOpts(opts.ItemStyle{
				Color: color,
			}),
		)
	}
	fmt.Println()

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return line.Render(f)
}

// generateSSIMByParamsChart creates a chart comparing SSIM values across parameters
// X-axis: BlockSize×D1D2 combinations
// Y-axis: SSIM values (median and average)
func generateSSIMByParamsChart(results []*db.DetailedResult, outputPath string) error {
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
			Title:    "SSIM Comparison by Parameters",
			Subtitle: "Comparing SSIM values (median and average) across BlockSize×D1D2 combinations",
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
