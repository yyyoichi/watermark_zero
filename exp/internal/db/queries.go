package db

import (
	"database/sql"
	"fmt"
)

// DetailedResult contains all joined information for a result
type DetailedResult struct {
	ID int64

	// Image info
	ImageURI    string
	Width       int
	Height      int
	ImageWidth  int
	ImageHeight int

	// Parameters
	BlockShapeH int
	BlockShapeW int
	D1          int
	D2          int

	// Mark info
	ECCAlgo      string
	EncodedSize  int
	OriginalSize int

	// Metrics
	EmbedCount      float64
	TotalBlocks     int
	EncodedAccuracy float64
	DecodedAccuracy float64
	Success         bool
	SSIM            float64

	// Paths
	OriginalImagePath string
	EmbedImagePath    string
}

// QueryDetailed executes a query on the results_detailed view
func (d *DB) QueryDetailed(query string, args ...interface{}) ([]*DetailedResult, error) {
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	var results []*DetailedResult
	for rows.Next() {
		var r DetailedResult
		err := rows.Scan(
			&r.ID,
			&r.ImageURI,
			&r.Width,
			&r.Height,
			&r.BlockShapeH,
			&r.BlockShapeW,
			&r.D1,
			&r.D2,
			&r.ECCAlgo,
			&r.EncodedSize,
			&r.OriginalSize,
			&r.EmbedCount,
			&r.TotalBlocks,
			&r.EncodedAccuracy,
			&r.DecodedAccuracy,
			&r.Success,
			&r.SSIM,
			&r.OriginalImagePath,
			&r.EmbedImagePath,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		r.ImageWidth = r.Width
		r.ImageHeight = r.Height
		results = append(results, &r)
	}
	return results, rows.Err()
}

// GetSuccessfulResults returns successful results with SSIM above threshold
func (d *DB) GetSuccessfulResults(minSSIM float64) ([]*DetailedResult, error) {
	return d.QueryDetailed(`
		SELECT * FROM results_detailed
		WHERE success = 1 AND ssim >= ?
		ORDER BY ssim DESC
	`, minSSIM)
}

// GetResultsByEmbedCount returns results within embed count range
func (d *DB) GetResultsByEmbedCount(minCount, maxCount float64) ([]*DetailedResult, error) {
	return d.QueryDetailed(`
		SELECT * FROM results_detailed
		WHERE embed_count BETWEEN ? AND ?
		ORDER BY embed_count
	`, minCount, maxCount)
}

// GetResultsByImageSize returns results for specific image dimensions
func (d *DB) GetResultsByImageSize(width, height int) ([]*DetailedResult, error) {
	return d.QueryDetailed(`
		SELECT * FROM results_detailed
		WHERE width = ? AND height = ?
		ORDER BY success DESC, ssim DESC
	`, width, height)
}

// GetResultsByD1D2 returns results for specific D1/D2 parameters
func (d *DB) GetResultsByD1D2(d1, d2 int) ([]*DetailedResult, error) {
	return d.QueryDetailed(`
		SELECT * FROM results_detailed
		WHERE d1 = ? AND d2 = ?
		ORDER BY success DESC, ssim DESC
	`, d1, d2)
}

// ParameterStats holds statistics for a parameter combination
type ParameterStats struct {
	BlockShapeH int
	BlockShapeW int
	D1          int
	D2          int
	TotalTests  int
	Successes   int
	SuccessRate float64
	AvgSSIM     float64
	AvgAccuracy float64
}

// GetBestParameters returns parameter combinations with best success rate
func (d *DB) GetBestParameters(minSuccessRate float64) ([]*ParameterStats, error) {
	rows, err := d.db.Query(`
		SELECT 
			block_shape_h, block_shape_w, d1, d2,
			COUNT(*) as total_tests,
			SUM(CASE WHEN success THEN 1 ELSE 0 END) as successes,
			AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END) as success_rate,
			AVG(ssim) as avg_ssim,
			AVG(decoded_accuracy) as avg_accuracy
		FROM results_detailed
		GROUP BY block_shape_h, block_shape_w, d1, d2
		HAVING success_rate >= ?
		ORDER BY success_rate DESC, avg_ssim DESC
	`, minSuccessRate)
	if err != nil {
		return nil, fmt.Errorf("failed to query best parameters: %w", err)
	}
	defer rows.Close()

	var stats []*ParameterStats
	for rows.Next() {
		var s ParameterStats
		err := rows.Scan(
			&s.BlockShapeH, &s.BlockShapeW, &s.D1, &s.D2,
			&s.TotalTests, &s.Successes, &s.SuccessRate,
			&s.AvgSSIM, &s.AvgAccuracy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		stats = append(stats, &s)
	}
	return stats, rows.Err()
}

// ImageSizeStats holds statistics for an image size
type ImageSizeStats struct {
	Width       int
	Height      int
	TotalTests  int
	Successes   int
	SuccessRate float64
	AvgSSIM     float64
	AvgAccuracy float64
}

// GetImageSizeStats returns statistics grouped by image size
func (d *DB) GetImageSizeStats() ([]*ImageSizeStats, error) {
	rows, err := d.db.Query(`
		SELECT 
			width, height,
			COUNT(*) as total_tests,
			SUM(CASE WHEN success THEN 1 ELSE 0 END) as successes,
			AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END) as success_rate,
			AVG(ssim) as avg_ssim,
			AVG(decoded_accuracy) as avg_accuracy
		FROM results_detailed
		GROUP BY width, height
		ORDER BY width, height
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query image size stats: %w", err)
	}
	defer rows.Close()

	var stats []*ImageSizeStats
	for rows.Next() {
		var s ImageSizeStats
		err := rows.Scan(
			&s.Width, &s.Height,
			&s.TotalTests, &s.Successes, &s.SuccessRate,
			&s.AvgSSIM, &s.AvgAccuracy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		stats = append(stats, &s)
	}
	return stats, rows.Err()
}

// EmbedCountStats holds statistics for an embed count range
type EmbedCountStats struct {
	EmbedCountRange string
	TotalTests      int
	Successes       int
	SuccessRate     float64
	AvgSSIM         float64
}

// GetEmbedCountStats returns statistics grouped by embed count ranges
func (d *DB) GetEmbedCountStats() ([]*EmbedCountStats, error) {
	rows, err := d.db.Query(`
		SELECT 
			CASE 
				WHEN embed_count < 1 THEN '0-1'
				WHEN embed_count < 2 THEN '1-2'
				WHEN embed_count < 4 THEN '2-4'
				WHEN embed_count < 6 THEN '4-6'
				WHEN embed_count < 8 THEN '6-8'
				ELSE '8+'
			END as range,
			COUNT(*) as total_tests,
			SUM(CASE WHEN success THEN 1 ELSE 0 END) as successes,
			AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END) as success_rate,
			AVG(ssim) as avg_ssim
		FROM results_detailed
		GROUP BY range
		ORDER BY 
			CASE range
				WHEN '0-1' THEN 1
				WHEN '1-2' THEN 2
				WHEN '2-4' THEN 3
				WHEN '4-6' THEN 4
				WHEN '6-8' THEN 5
				ELSE 6
			END
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query embed count stats: %w", err)
	}
	defer rows.Close()

	var stats []*EmbedCountStats
	for rows.Next() {
		var s EmbedCountStats
		err := rows.Scan(
			&s.EmbedCountRange,
			&s.TotalTests, &s.Successes, &s.SuccessRate,
			&s.AvgSSIM,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %w", err)
		}
		stats = append(stats, &s)
	}
	return stats, rows.Err()
}

// ExecuteRawQuery executes a raw SQL query and returns rows
func (d *DB) ExecuteRawQuery(query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.Query(query, args...)
}
