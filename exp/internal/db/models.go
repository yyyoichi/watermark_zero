package db

type (
	// Image represents source image URL
	Image struct {
		ID  int64
		URI string // Unique constraint
	}

	// ImageSize represents resized dimensions
	ImageSize struct {
		ID      int64
		ImageID int64
		Width   int
		Height  int
		// Unique constraint on (ImageID, Width, Height)
	}

	// Mark represents original watermark data
	Mark struct {
		ID   int64
		Mark []byte // Use []byte for binary data
		Size int    // Bit length
	}

	// ECCMark represents encoded watermark
	ECCMark struct {
		ID       int64
		MarkID   int64
		Encoded  []byte // Use []byte for binary data
		Size     int    // Bit length
		AlgoName string
		// Unique constraint on (MarkID, AlgoName)
	}

	// MarkParam represents watermarking parameters
	MarkParam struct {
		ID          int64
		BlockShapeH int
		BlockShapeW int
		D1          int
		D2          int
		// Unique constraint on (BlockShapeH, BlockShapeW, D1, D2)
	}

	// Result represents test outcome
	Result struct {
		ID          int64
		ImageSizeID int64
		ECCMarkID   int64
		MarkParamID int64

		// Paths
		OriginalImagePath string
		EmbedImagePath    string

		// Computed fields (can be calculated from relations)
		EmbedCount  float64 // TotalBlocks / EncodedSize
		TotalBlocks int     // (Width/BlockW) * (Height/BlockH)

		// Evaluation metrics
		EncodedAccuracy float64
		DecodedAccuracy float64
		Success         bool
		SSIM            float64

		// Unique constraint on (ImageSizeID, ECCMarkID, MarkParamID)
	}
)
