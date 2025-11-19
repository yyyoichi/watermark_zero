package main

import (
	"encoding/json"
	"exp/internal/db"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	dbPath := flag.String("db", "./tmp/optimize/jsons/optimize_results.db", "Path to database file")
	queryType := flag.String("query", "stats", "Query type: stats, best-params, image-sizes, embed-counts, successful, raw")
	minSSIM := flag.Float64("min-ssim", 0.95, "Minimum SSIM for successful results")
	minSuccessRate := flag.Float64("min-success", 0.8, "Minimum success rate for best params")
	rawSQL := flag.String("sql", "", "Raw SQL query to execute")

	flag.Parse()

	database, err := db.Open(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer database.Close()

	switch *queryType {
	case "stats":
		count, err := database.CountResults()
		if err != nil {
			log.Fatalf("Failed to count results: %v", err)
		}
		fmt.Printf("Total results: %d\n", count)

	case "best-params":
		stats, err := database.GetBestParameters(*minSuccessRate)
		if err != nil {
			log.Fatalf("Failed to get best parameters: %v", err)
		}
		printJSON(stats)

	case "image-sizes":
		stats, err := database.GetImageSizeStats()
		if err != nil {
			log.Fatalf("Failed to get image size stats: %v", err)
		}
		printJSON(stats)

	case "embed-counts":
		stats, err := database.GetEmbedCountStats()
		if err != nil {
			log.Fatalf("Failed to get embed count stats: %v", err)
		}
		printJSON(stats)

	case "successful":
		results, err := database.GetSuccessfulResults(*minSSIM)
		if err != nil {
			log.Fatalf("Failed to get successful results: %v", err)
		}
		printJSON(results)

	case "raw":
		if *rawSQL == "" {
			log.Fatal("Please provide SQL query with -sql flag")
		}
		rows, err := database.ExecuteRawQuery(*rawSQL)
		if err != nil {
			log.Fatalf("Failed to execute query: %v", err)
		}
		defer rows.Close()

		// Get column names
		cols, err := rows.Columns()
		if err != nil {
			log.Fatalf("Failed to get columns: %v", err)
		}

		// Print results
		fmt.Println("Columns:", cols)
		for rows.Next() {
			// Create slice for scanning
			values := make([]interface{}, len(cols))
			valuePtrs := make([]interface{}, len(cols))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			if err := rows.Scan(valuePtrs...); err != nil {
				log.Fatalf("Failed to scan row: %v", err)
			}

			// Print row
			for i, col := range cols {
				fmt.Printf("%s: %v\n", col, values[i])
			}
			fmt.Println("---")
		}

	default:
		log.Fatalf("Unknown query type: %s", *queryType)
	}
}

func printJSON(v interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		log.Fatalf("Failed to encode JSON: %v", err)
	}
}
