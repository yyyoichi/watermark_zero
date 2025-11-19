package db

import (
	"database/sql"
	"fmt"
)

// InsertImage inserts or gets an existing image by URI
func (d *DB) InsertImage(uri string) (int64, error) {
	// Try to get existing
	var id int64
	err := d.db.QueryRow("SELECT id FROM images WHERE uri = ?", uri).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query image: %w", err)
	}

	// Insert new
	result, err := d.db.Exec("INSERT INTO images (uri) VALUES (?)", uri)
	if err != nil {
		return 0, fmt.Errorf("failed to insert image: %w", err)
	}
	return result.LastInsertId()
}

// InsertImageSize inserts or gets an existing image size
func (d *DB) InsertImageSize(imageID int64, width, height int) (int64, error) {
	// Try to get existing
	var id int64
	err := d.db.QueryRow(
		"SELECT id FROM image_sizes WHERE image_id = ? AND width = ? AND height = ?",
		imageID, width, height,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query image size: %w", err)
	}

	// Insert new
	result, err := d.db.Exec(
		"INSERT INTO image_sizes (image_id, width, height) VALUES (?, ?, ?)",
		imageID, width, height,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert image size: %w", err)
	}
	return result.LastInsertId()
}

// InsertMark inserts or gets an existing mark
func (d *DB) InsertMark(mark []byte, size int) (int64, error) {
	// Try to get existing
	var id int64
	err := d.db.QueryRow(
		"SELECT id FROM marks WHERE mark = ? AND size = ?",
		mark, size,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query mark: %w", err)
	}

	// Insert new
	result, err := d.db.Exec(
		"INSERT INTO marks (mark, size) VALUES (?, ?)",
		mark, size,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert mark: %w", err)
	}
	return result.LastInsertId()
}

// InsertECCMark inserts or gets an existing ECC mark
func (d *DB) InsertECCMark(markID int64, encoded []byte, size int, algoName string) (int64, error) {
	// Try to get existing
	var id int64
	err := d.db.QueryRow(
		"SELECT id FROM ecc_marks WHERE mark_id = ? AND algo_name = ?",
		markID, algoName,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query ecc mark: %w", err)
	}

	// Insert new
	result, err := d.db.Exec(
		"INSERT INTO ecc_marks (mark_id, encoded, size, algo_name) VALUES (?, ?, ?, ?)",
		markID, encoded, size, algoName,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert ecc mark: %w", err)
	}
	return result.LastInsertId()
}

// InsertMarkParam inserts or gets existing mark parameters
func (d *DB) InsertMarkParam(blockShapeH, blockShapeW, d1, d2 int) (int64, error) {
	// Try to get existing
	var id int64
	err := d.db.QueryRow(
		"SELECT id FROM mark_params WHERE block_shape_h = ? AND block_shape_w = ? AND d1 = ? AND d2 = ?",
		blockShapeH, blockShapeW, d1, d2,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query mark param: %w", err)
	}

	// Insert new
	result, err := d.db.Exec(
		"INSERT INTO mark_params (block_shape_h, block_shape_w, d1, d2) VALUES (?, ?, ?, ?)",
		blockShapeH, blockShapeW, d1, d2,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert mark param: %w", err)
	}
	return result.LastInsertId()
}

// InsertResult inserts a result (or updates if already exists)
func (d *DB) InsertResult(result *Result) (int64, error) {
	// Check if result already exists
	var existingID int64
	err := d.db.QueryRow(
		"SELECT id FROM results WHERE image_size_id = ? AND ecc_mark_id = ? AND mark_param_id = ?",
		result.ImageSizeID, result.ECCMarkID, result.MarkParamID,
	).Scan(&existingID)

	if err == nil {
		// Update existing
		_, err = d.db.Exec(`
			UPDATE results SET
				original_image_path = ?,
				embed_image_path = ?,
				embed_count = ?,
				total_blocks = ?,
				encoded_accuracy = ?,
				decoded_accuracy = ?,
				success = ?,
				ssim = ?
			WHERE id = ?`,
			result.OriginalImagePath,
			result.EmbedImagePath,
			result.EmbedCount,
			result.TotalBlocks,
			result.EncodedAccuracy,
			result.DecodedAccuracy,
			result.Success,
			result.SSIM,
			existingID,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to update result: %w", err)
		}
		return existingID, nil
	}

	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query existing result: %w", err)
	}

	// Insert new
	res, err := d.db.Exec(`
		INSERT INTO results (
			image_size_id, ecc_mark_id, mark_param_id,
			original_image_path, embed_image_path,
			embed_count, total_blocks,
			encoded_accuracy, decoded_accuracy, success, ssim
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		result.ImageSizeID,
		result.ECCMarkID,
		result.MarkParamID,
		result.OriginalImagePath,
		result.EmbedImagePath,
		result.EmbedCount,
		result.TotalBlocks,
		result.EncodedAccuracy,
		result.DecodedAccuracy,
		result.Success,
		result.SSIM,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert result: %w", err)
	}
	return res.LastInsertId()
}

// GetImage retrieves an image by ID
func (d *DB) GetImage(id int64) (*Image, error) {
	var img Image
	err := d.db.QueryRow("SELECT id, uri FROM images WHERE id = ?", id).Scan(&img.ID, &img.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	return &img, nil
}

// GetImageSize retrieves an image size by ID
func (d *DB) GetImageSize(id int64) (*ImageSize, error) {
	var size ImageSize
	err := d.db.QueryRow(
		"SELECT id, image_id, width, height FROM image_sizes WHERE id = ?", id,
	).Scan(&size.ID, &size.ImageID, &size.Width, &size.Height)
	if err != nil {
		return nil, fmt.Errorf("failed to get image size: %w", err)
	}
	return &size, nil
}

// GetMarkParam retrieves mark parameters by ID
func (d *DB) GetMarkParam(id int64) (*MarkParam, error) {
	var param MarkParam
	err := d.db.QueryRow(
		"SELECT id, block_shape_h, block_shape_w, d1, d2 FROM mark_params WHERE id = ?", id,
	).Scan(&param.ID, &param.BlockShapeH, &param.BlockShapeW, &param.D1, &param.D2)
	if err != nil {
		return nil, fmt.Errorf("failed to get mark param: %w", err)
	}
	return &param, nil
}

// ListResults retrieves all results
func (d *DB) ListResults() ([]*Result, error) {
	rows, err := d.db.Query(`
		SELECT id, image_size_id, ecc_mark_id, mark_param_id,
		       original_image_path, embed_image_path,
		       embed_count, total_blocks,
		       encoded_accuracy, decoded_accuracy, success, ssim
		FROM results
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query results: %w", err)
	}
	defer rows.Close()

	var results []*Result
	for rows.Next() {
		var r Result
		err := rows.Scan(
			&r.ID, &r.ImageSizeID, &r.ECCMarkID, &r.MarkParamID,
			&r.OriginalImagePath, &r.EmbedImagePath,
			&r.EmbedCount, &r.TotalBlocks,
			&r.EncodedAccuracy, &r.DecodedAccuracy, &r.Success, &r.SSIM,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}
		results = append(results, &r)
	}
	return results, rows.Err()
}

// CountResults counts total results
func (d *DB) CountResults() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM results").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count results: %w", err)
	}
	return count, nil
}
