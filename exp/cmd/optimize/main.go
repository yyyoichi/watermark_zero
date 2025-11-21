package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
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
		fmt.Println("3. Start HTTP server to view visualizations")
		fmt.Println("4. Exit")
		fmt.Print("\nSelect an option (1-4): ")

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

			fmt.Printf("\nStarting with: numImages=%d, offset=%d\n\n",
				numImages, offset)

			runMain(numImages, offset)
		case "2":
			fmt.Println("\n--- Visualizing Results ---")

			// Get JSON directory
			fmt.Printf("JSON directory (default: %s): ", TmpOptimizeJsonsDir)
			jsonDir, _ := reader.ReadString('\n')
			jsonDir = strings.TrimSpace(jsonDir)
			if jsonDir == "" {
				jsonDir = TmpOptimizeJsonsDir
			}

			// List JSON files in the directory
			jsonFiles, err := listJSONFiles(jsonDir)
			if err != nil {
				fmt.Printf("Error reading JSON directory: %v\n", err)
				continue
			}

			if len(jsonFiles) == 0 {
				fmt.Println("No JSON files found in the directory")
				continue
			}

			// Display JSON files with indices (newest first)
			fmt.Println("\nAvailable JSON files (newest first):")
			for i, file := range jsonFiles {
				fmt.Printf("  [%d] %s (modified: %s)\n", i+1, file.Name, file.ModTime.Format("2006-01-02 15:04:05"))
			}

			// Get file selection
			fmt.Printf("\nSelect a JSON file (1-%d, default: 1): ", len(jsonFiles))
			fileIndexStr, _ := reader.ReadString('\n')
			fileIndexStr = strings.TrimSpace(fileIndexStr)
			fileIndex := 1
			if fileIndexStr != "" {
				if val, err := strconv.Atoi(fileIndexStr); err == nil && val >= 1 && val <= len(jsonFiles) {
					fileIndex = val
				} else {
					fmt.Printf("Invalid selection, using default (1)\n")
				}
			}

			inputFile := jsonFiles[fileIndex-1].Path

			// Get output directory
			fmt.Printf("\nOutput directory for visualizations (default: %s): ", TmpOptimizeDir)
			outputDir, _ := reader.ReadString('\n')
			outputDir = strings.TrimSpace(outputDir)
			if outputDir == "" {
				outputDir = TmpOptimizeDir
			}

			fmt.Printf("\nVisualizing: inputFile=%s, outputDir=%s\n\n", inputFile, outputDir)

			visualizeMain(inputFile, outputDir)
		case "3":
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
		case "4":
			fmt.Println("Exiting...")
			closeDatabase()
			os.Exit(0)
		default:
			fmt.Println("Invalid option. Please select 1-4.")
		}
	}
}

func init() {
	// mkdir tmp directories
	os.MkdirAll(TmpOptimizeDir, 0755)
	os.MkdirAll(TmpOptimizeJsonsDir, 0755)
	os.MkdirAll(TmpOptimizeEmbeddedImagesDir, 0755)
}

// JSONFileInfo holds information about a JSON file
type JSONFileInfo struct {
	Name    string
	Path    string
	ModTime time.Time
}

// listJSONFiles lists all JSON files in a directory sorted by modification time (newest first)
func listJSONFiles(dir string) ([]JSONFileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var jsonFiles []JSONFileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		jsonFiles = append(jsonFiles, JSONFileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(dir, entry.Name()),
			ModTime: info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(jsonFiles, func(i, j int) bool {
		return jsonFiles[i].ModTime.After(jsonFiles[j].ModTime)
	})

	return jsonFiles, nil
}
