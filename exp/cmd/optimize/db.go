package main

import (
	"exp/internal/db"
	"exp/internal/images"
	"log"
	"math"
	"os"
	"path/filepath"
)

// Global database instance
var database *db.DB

// Database configuration
const dbFilename = "optimize_results.db"

var (
	EccAlgoShuffledGolay = "S-Golay"
	EccAlgoNoEcc         = "NoEcc"
)

func init() {
	// Initialize database
	dbDir := filepath.Join("/tmp/optimize-db")
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	dbPath := filepath.Join(dbDir, dbFilename)
	var err error
	database, err = db.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	log.Printf("Database initialized: %s\n", dbPath)

	// Insert uris
	{
		urls := images.ParseURLs()
		for _, url := range urls {
			_, err := database.InsertImage(url)
			if err != nil {
				log.Printf("Failed to insert image %s: %v", url, err)
			}
		}
	}
	// Insert sizes
	{
		// Golay standard sizes
		// wmarkzero: 664(83x8)bits -> Gloy (664+11)/12*23 = 1288bits
		base := 1288
		// embed count 1 ~ 30
		count := 30
		var defaultImageSizes = [][]int{}
		// 8x8 block base size
		for i := 1; i <= count; i++ {
			totalBits := base * i
			v := math.Ceil(math.Sqrt(float64(totalBits * 64)))
			size := int(v+7) / 8 * 8 // align to 8
			defaultImageSizes = append(defaultImageSizes, []int{size, size})
		}
		for _, size := range defaultImageSizes {
			_, err := database.InsertImageSize(size[0], size[1])
			if err != nil {
				log.Printf("Failed to insert image size %dx%d: %v", size[0], size[1], err)
			}
		}
	}
	// Insert marks
	{
		// wzeromark
		mark := make([]byte, 83)
		for i := range mark {
			mark[i] = uint8(i*2 + 1) // Dummy data
		}
		_, err := database.InsertMark(mark)
		if err != nil {
			log.Printf("Failed to insert mark: %v", err)
		}
	}
	// Insert ecc algos
	{
		_, err := database.InsertMarkEccAlgo(EccAlgoShuffledGolay)
		if err != nil {
			log.Printf("Failed to insert ECC algo %s: %v", EccAlgoShuffledGolay, err)
		}
		_, err = database.InsertMarkEccAlgo(EccAlgoNoEcc)
		if err != nil {
			log.Printf("Failed to insert ECC algo %s: %v", EccAlgoNoEcc, err)
		}
	}
	// Insert mark params
	{
		var shapes = [][]int{
			{8, 8},
			{6, 6},
			{4, 4},
		}

		var d1d2Pairs = [][]int{
			{21, 11}, {21, 9}, {21, 7}, {21, 5}, {21, 3},
			{19, 11}, {19, 9}, {19, 7}, {19, 5}, {19, 3},
			{17, 11}, {17, 9}, {17, 7}, {17, 5}, {17, 3},
			{15, 11}, {15, 9}, {15, 7}, {15, 5}, {15, 3},
		}

		for _, bs := range shapes {
			for _, d1d2 := range d1d2Pairs {
				_, err := database.InsertMarkParam(bs[1], bs[0], d1d2[0], d1d2[1])
				if err != nil {
					log.Printf("Failed to insert mark param (bs=%dx%d, d1d2=%dx%d): %v", bs[0], bs[1], d1d2[0], d1d2[1], err)
				}
			}
		}
	}
}

// closeDatabase should be called on program exit
func closeDatabase() {
	if database != nil {
		if err := database.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
}
