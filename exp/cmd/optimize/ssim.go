package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// calculateSSIM uses ffmpeg to calculate SSIM between two images
func calculateSSIM(originalPath, embeddedPath string) (float64, error) {
	// Check if files exist
	if _, err := os.Stat(originalPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("original image not found: %s", originalPath)
	}
	if _, err := os.Stat(embeddedPath); os.IsNotExist(err) {
		return 0, fmt.Errorf("embedded image not found: %s", embeddedPath)
	}

	// ffmpeg command to calculate SSIM
	cmd := exec.Command("ffmpeg",
		"-i", originalPath,
		"-i", embeddedPath,
		"-lavfi", "ssim",
		"-f", "null",
		"-")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("ffmpeg error: %w, output: %s", err, string(output))
	}

	// Parse SSIM value from output
	// Looking for line like: "[Parsed_ssim_0 @ 0x...] SSIM Y:0.999999 U:0.999999 V:0.999999 All:0.999999 (60.00 dB)"
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		if strings.Contains(line, "SSIM") && strings.Contains(line, "All:") {
			// Extract SSIM All value
			parts := strings.Split(line, "All:")
			if len(parts) < 2 {
				continue
			}
			valuePart := strings.Split(parts[1], " ")[0]
			ssim, err := strconv.ParseFloat(valuePart, 64)
			if err != nil {
				return 0, fmt.Errorf("failed to parse SSIM All value: %w", err)
			}
			return ssim, nil
		}
	}

	return 0, fmt.Errorf("SSIM All value not found in ffmpeg output")
}
