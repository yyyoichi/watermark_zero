package watermark

type Option func(*Watermark) error

// WithBlockShape divides the image into blocks of the specified size for processing.
// For example, for a 600x480 image with an 8x6 block shape, it creates 75 horizontal
// and 80 vertical blocks.
//
// Block shapes must be specified with even numbers, with a minimum size of 4x4.
// If odd numbers are provided, they are automatically rounded up to the next even number.
// If values smaller than 4 are provided, they are set to 4.
func WithBlockShape(width, height int) Option {
	return func(w *Watermark) error {
		s := newBlockShape(width, height)
		w.blockShape = &s
		return nil
	}
}

// WithD1 specifies the d1 parameter for watermark embedding and extraction.
// Larger values increase noise but improve robustness.
// This option has less computational cost than WithD1D2 but may have lower robustness in comparison.
func WithD1(d1 int) Option {
	return func(w *Watermark) error {
		if err := w.setEmbedD1(d1); err != nil {
			return err
		}
		if err := w.setExtractD1(d1); err != nil {
			return err
		}
		return nil
	}
}

// WithD1D2 specifies both d1 and d2 parameters for watermark embedding and extraction.
// Larger values increase noise but improve robustness.
// This option has higher computational cost than WithD1 but may provide better robustness.
func WithD1D2(d1, d2 int) Option {
	return func(w *Watermark) error {
		if err := w.setEmbedD1D2(d1, d2); err != nil {
			return err
		}
		if err := w.setExtractD1D2(d1, d2); err != nil {
			return err
		}
		return nil
	}
}
