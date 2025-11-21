package main

import (
	"exp/internal/db"
	"log"
	"path/filepath"
)

// Global database instance
var database *db.DB

// Database configuration
const dbFilename = "optimize_results.db"

func init() {
	// Initialize database
	dbPath := filepath.Join(TmpOptimizeJsonsDir, dbFilename)

	var err error
	database, err = db.Open(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	log.Printf("Database initialized: %s\n", dbPath)
}

// closeDatabase should be called on program exit
func closeDatabase() {
	if database != nil {
		if err := database.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
}
