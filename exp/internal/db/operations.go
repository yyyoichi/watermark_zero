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

// InsertImageSize inserts or gets an existing image size (independent of source image)
func (d *DB) InsertImageSize(width, height int) (int64, error) {
	// Try to get existing
	var id int64
	err := d.db.QueryRow(
		"SELECT id FROM image_sizes WHERE width = ? AND height = ?",
		width, height,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query image size: %w", err)
	}

	// Insert new
	result, err := d.db.Exec(
		"INSERT INTO image_sizes (width, height) VALUES (?, ?)",
		width, height,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert image size: %w", err)
	}
	return result.LastInsertId()
}

// InsertMark inserts or gets an existing mark
func (d *DB) InsertMark(mark []byte) (int64, error) {
	// Try to get existing
	var id int64
	err := d.db.QueryRow(
		"SELECT id FROM marks WHERE mark = ?",
		mark,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query mark: %w", err)
	}

	// Insert new
	result, err := d.db.Exec(
		"INSERT INTO marks (mark) VALUES (?)",
		mark,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert mark: %w", err)
	}
	return result.LastInsertId()
}

// InsertMarkEccAlgo inserts or gets an existing Mark ECC algorithm
func (d *DB) InsertMarkEccAlgo(algoName string) (int64, error) {
	// Try to get existing
	var id int64
	err := d.db.QueryRow(
		"SELECT id FROM mark_ecc_algos WHERE algo_name = ?",
		algoName,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to query mark ecc algo: %w", err)
	}

	// Insert new
	result, err := d.db.Exec(
		"INSERT INTO mark_ecc_algos (algo_name) VALUES (?)",
		algoName,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert mark ecc algo: %w", err)
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

// ResultExists checks if a result already exists for the given parameters
// Returns the result ID if exists, 0 if not found
func (d *DB) ResultExists(imageID, imageSizeID, markID, markEccAlgoID, markParamID int64) (int64, error) {
	var id int64
	err := d.db.QueryRow(
		"SELECT id FROM results WHERE image_id = ? AND image_size_id = ? AND mark_id = ? AND mark_ecc_algo_id = ? AND mark_param_id = ?",
		imageID, imageSizeID, markID, markEccAlgoID, markParamID,
	).Scan(&id)

	if err == sql.ErrNoRows {
		return 0, nil // Not found
	}
	if err != nil {
		return 0, fmt.Errorf("failed to query result existence: %w", err)
	}

	return id, nil
}

// InsertResult inserts a result (or updates if already exists)
func (d *DB) InsertResult(result *Result) (int64, error) {
	// Check if result already exists
	var existingID int64
	err := d.db.QueryRow(
		"SELECT id FROM results WHERE image_id = ? AND image_size_id = ? AND mark_id = ? AND mark_ecc_algo_id = ? AND mark_param_id = ?",
		result.ImageID, result.ImageSizeID, result.MarkID, result.MarkEccAlgoID, result.MarkParamID,
	).Scan(&existingID)

	if err == nil {
		// Update existing
		_, err = d.db.Exec(`
			UPDATE results SET
				embed_count = ?,
				total_blocks = ?,
				encoded_accuracy = ?,
				decoded_accuracy = ?,
				success = ?,
				ssim = ?
			WHERE id = ?`,
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
			image_id, image_size_id, mark_id, mark_ecc_algo_id, mark_param_id,
			embed_count, total_blocks,
			encoded_accuracy, decoded_accuracy, success, ssim
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		result.ImageID,
		result.ImageSizeID,
		result.MarkID,
		result.MarkEccAlgoID,
		result.MarkParamID,
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
		"SELECT id, width, height FROM image_sizes WHERE id = ?", id,
	).Scan(&size.ID, &size.Width, &size.Height)
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
		SELECT id, image_id, image_size_id, mark_id, mark_ecc_algo_id, mark_param_id,
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
			&r.ID, &r.ImageID, &r.ImageSizeID, &r.MarkID, &r.MarkEccAlgoID, &r.MarkParamID,
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

// GetImageSizeByID retrieves an image size by ID
func (d *DB) GetImageSizeByID(id int64) (*ImageSize, error) {
	var s ImageSize
	err := d.db.QueryRow(
		"SELECT id, width, height FROM image_sizes WHERE id = ?",
		id,
	).Scan(&s.ID, &s.Width, &s.Height)
	if err != nil {
		return nil, fmt.Errorf("failed to get image size: %w", err)
	}
	return &s, nil
}

// GetMarkParamByID retrieves a mark param by ID
func (d *DB) GetMarkParamByID(id int64) (*MarkParam, error) {
	var mp MarkParam
	err := d.db.QueryRow(
		"SELECT id, block_shape_h, block_shape_w, d1, d2 FROM mark_params WHERE id = ?",
		id,
	).Scan(&mp.ID, &mp.BlockShapeH, &mp.BlockShapeW, &mp.D1, &mp.D2)
	if err != nil {
		return nil, fmt.Errorf("failed to get mark param: %w", err)
	}
	return &mp, nil
}

// GetMarkEccAlgoByID retrieves a mark ECC algo by ID
func (d *DB) GetMarkEccAlgoByID(id int64) (*MarkEccAlgo, error) {
	var mea MarkEccAlgo
	err := d.db.QueryRow(
		"SELECT id, algo_name FROM mark_ecc_algos WHERE id = ?",
		id,
	).Scan(&mea.ID, &mea.AlgoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get mark ecc algo: %w", err)
	}
	return &mea, nil
}

// ListImageSizes retrieves all image sizes
func (d *DB) ListImageSizes() ([]*ImageSize, error) {
	rows, err := d.db.Query(`
		SELECT id, width, height
		FROM image_sizes
		ORDER BY width, height
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query image sizes: %w", err)
	}
	defer rows.Close()

	var sizes []*ImageSize
	for rows.Next() {
		var s ImageSize
		err := rows.Scan(&s.ID, &s.Width, &s.Height)
		if err != nil {
			return nil, fmt.Errorf("failed to scan image size: %w", err)
		}
		sizes = append(sizes, &s)
	}
	return sizes, rows.Err()
}

// ListMarkParams retrieves all mark parameters
func (d *DB) ListMarkParams() ([]*MarkParam, error) {
	rows, err := d.db.Query(`
		SELECT id, block_shape_h, block_shape_w, d1, d2
		FROM mark_params
		ORDER BY block_shape_w, block_shape_h, d1, d2
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query mark params: %w", err)
	}
	defer rows.Close()

	var params []*MarkParam
	for rows.Next() {
		var mp MarkParam
		err := rows.Scan(&mp.ID, &mp.BlockShapeH, &mp.BlockShapeW, &mp.D1, &mp.D2)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mark param: %w", err)
		}
		params = append(params, &mp)
	}
	return params, rows.Err()
}

// ListMarkEccAlgos retrieves all mark ECC algorithms
func (d *DB) ListMarkEccAlgos() ([]*MarkEccAlgo, error) {
	rows, err := d.db.Query(`
		SELECT id, algo_name
		FROM mark_ecc_algos
		ORDER BY algo_name
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query mark ecc algos: %w", err)
	}
	defer rows.Close()

	var algos []*MarkEccAlgo
	for rows.Next() {
		var mea MarkEccAlgo
		err := rows.Scan(&mea.ID, &mea.AlgoName)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mark ecc algo: %w", err)
		}
		algos = append(algos, &mea)
	}
	return algos, rows.Err()
}

// ListMarks retrieves all marks
func (d *DB) ListMarks() ([]*Mark, error) {
	rows, err := d.db.Query(`
		SELECT id, mark
		FROM marks
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query marks: %w", err)
	}
	defer rows.Close()

	var marks []*Mark
	for rows.Next() {
		var m Mark
		err := rows.Scan(&m.ID, &m.Mark)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mark: %w", err)
		}
		marks = append(marks, &m)
	}
	return marks, rows.Err()
}
