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
	// TmpOptimizeEmbeddedImagesDir is the directory for embedded image cache
	TmpOptimizeEmbeddedImagesDir = "/tmp/optimize-embedded-images"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n=== Watermark Optimization Tool ===")
		fmt.Println("1. Run optimization experiments (save to Database)")
		fmt.Println("2. Visualize results from Database")
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
			fmt.Println("\n--- Visualizing Results from Database ---")

			// Get output directory
			fmt.Printf("Output directory for visualizations (default: %s): ", TmpOptimizeDir)
			outputDir, _ := reader.ReadString('\n')
			outputDir = strings.TrimSpace(outputDir)
			if outputDir == "" {
				outputDir = TmpOptimizeDir
			}

			fmt.Printf("\nGenerating visualizations to: %s\n\n", outputDir)

			visualizeMain(outputDir)
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
	os.MkdirAll(TmpOptimizeEmbeddedImagesDir, 0755)
}
