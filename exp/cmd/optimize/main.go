package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	// TmpOptimizeDir is the base directory for optimization outputs
	TmpOptimizeDir = "/tmp/optimize"
	// TmpOptimizeJsonsDir is the directory for JSON output files
	TmpOptimizeJsonsDir = "/tmp/optimize-jsons"
	// TmpOptimizeEmbeddedImagesDir is the directory for embedded image cache
	TmpOptimizeEmbeddedImagesDir = "/tmp/optimize-embedded-images"
)

type (
	DataJsonFormat struct {
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
	OptimizeResult struct {
		OriginalImagePath string
		EmbedImagePath    string

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
		SSIM            float64
	}
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n=== Watermark Optimization Tool ===")
		fmt.Println("1. Run optimization experiments (save to JSON)")
		fmt.Println("2. Visualize results from JSON file")
		fmt.Println("3. Analyze image quality degradation (SSIM vs Success Rate)")
		fmt.Println("4. Start HTTP server to view visualizations")
		fmt.Println("5. Exit")
		fmt.Print("\nSelect an option (1-5): ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch input {
		case "1":
			fmt.Println("\n--- Running Optimization Experiments ---")

			// Get number of images
			fmt.Print("Number of images to test (default: 10): ")
			numImagesStr, _ := reader.ReadString('\n')
			numImagesStr = strings.TrimSpace(numImagesStr)
			numImages := 10
			if numImagesStr != "" {
				if val, err := strconv.Atoi(numImagesStr); err == nil {
					numImages = val
				}
			}

			// Get offset
			fmt.Print("Offset to start from (default: 0): ")
			offsetStr, _ := reader.ReadString('\n')
			offsetStr = strings.TrimSpace(offsetStr)
			offset := 0
			if offsetStr != "" {
				if val, err := strconv.Atoi(offsetStr); err == nil {
					offset = val
				}
			}

			// Get target embed low
			fmt.Print("Target embed count lower bound (default: 1.0): ")
			embedLowStr, _ := reader.ReadString('\n')
			embedLowStr = strings.TrimSpace(embedLowStr)
			targetEmbedLow := 1.0
			if embedLowStr != "" {
				if val, err := strconv.ParseFloat(embedLowStr, 64); err == nil {
					targetEmbedLow = val
				}
			}

			// Get target embed high
			fmt.Print("Target embed count upper bound (default: 6.0): ")
			embedHighStr, _ := reader.ReadString('\n')
			embedHighStr = strings.TrimSpace(embedHighStr)
			targetEmbedHigh := 6.0
			if embedHighStr != "" {
				if val, err := strconv.ParseFloat(embedHighStr, 64); err == nil {
					targetEmbedHigh = val
				}
			}

			fmt.Printf("\nStarting with: numImages=%d, offset=%d, embedRange=%.1f-%.1f\n\n",
				numImages, offset, targetEmbedLow, targetEmbedHigh)

			runMain(numImages, offset, targetEmbedLow, targetEmbedHigh)
		case "2":
			fmt.Println("\n--- Visualizing Results ---")

			// Get input file path
			fmt.Print("JSON file path to visualize: ")
			inputFile, _ := reader.ReadString('\n')
			inputFile = strings.TrimSpace(inputFile)

			if inputFile == "" {
				fmt.Println("Error: Input file path is required")
				os.Exit(1)
			}

			// Get output directory
			fmt.Printf("Output directory for visualizations (default: %s): ", TmpOptimizeDir)
			outputDir, _ := reader.ReadString('\n')
			outputDir = strings.TrimSpace(outputDir)
			if outputDir == "" {
				outputDir = TmpOptimizeDir
			}

			fmt.Printf("\nVisualizing: inputFile=%s, outputDir=%s\n\n", inputFile, outputDir)

			visualizeMain(inputFile, outputDir)
		case "3":
			fmt.Println("\n--- Analyzing Image Quality Degradation ---")

			// Get input file path
			fmt.Print("JSON file path to analyze: ")
			inputFile, _ := reader.ReadString('\n')
			inputFile = strings.TrimSpace(inputFile)

			if inputFile == "" {
				fmt.Println("Error: Input file path is required")
				os.Exit(1)
			}

			// Get output directory
			fmt.Printf("Output directory for quality analysis (default: %s): ", TmpOptimizeDir)
			outputDir, _ := reader.ReadString('\n')
			outputDir = strings.TrimSpace(outputDir)
			if outputDir == "" {
				outputDir = TmpOptimizeDir
			}

			fmt.Printf("\nAnalyzing quality: inputFile=%s, outputDir=%s\n\n", inputFile, outputDir)

			qualityMain(inputFile, outputDir)
		case "4":
			fmt.Println("\n--- Starting HTTP Server ---")

			// Get server directory
			fmt.Printf("Directory to serve (default: %s): ", TmpOptimizeDir)
			serverDir, _ := reader.ReadString('\n')
			serverDir = strings.TrimSpace(serverDir)
			if serverDir == "" {
				serverDir = TmpOptimizeDir
			}

			fmt.Println("Server will start at http://localhost:8080")
			fmt.Println("Press Ctrl+C to stop the server")
			fmt.Println()

			startHTTPServer(serverDir)
		case "5":
			fmt.Println("Exiting...")
			os.Exit(0)
		default:
			fmt.Println("Invalid option. Please select 1-5.")
		}
	}
}

func init() {
	// mkdir tmp directories
	os.MkdirAll(TmpOptimizeDir, 0755)
	os.MkdirAll(TmpOptimizeJsonsDir, 0755)
	os.MkdirAll(TmpOptimizeEmbeddedImagesDir, 0755)
}
