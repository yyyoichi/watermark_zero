package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

func startHTTPServer(serverDir string) {
	// Use default directory if not specified
	if serverDir == "" {
		serverDir = "/tmp/optimize"
	}

	// Check if directory exists
	if _, err := os.Stat(serverDir); os.IsNotExist(err) {
		log.Printf("Directory %s does not exist. Creating it...\n", serverDir)
		if err := os.MkdirAll(serverDir, 0755); err != nil {
			log.Fatalf("Failed to create directory: %v", err)
		}
	}

	// Create command to start Python HTTP server
	cmd := exec.Command("python3", "-m", "http.server", "8080")
	cmd.Dir = serverDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the server
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}

	fmt.Printf("HTTP server started on http://localhost:8080\n")
	fmt.Printf("Serving files from: %s\n", serverDir)
	fmt.Println("Press Ctrl+C to stop the server...")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal
	<-sigChan

	fmt.Println("\n\nShutting down server...")

	// Kill the server process
	if err := cmd.Process.Kill(); err != nil {
		log.Printf("Failed to kill server process: %v", err)
	}

	// Wait for the process to exit
	_ = cmd.Wait()

	fmt.Println("Server stopped.")
}
