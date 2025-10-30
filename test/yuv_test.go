package test

import (
	_ "embed"
	"encoding/json"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyyoichi/watermark_zero/internal/yuv"
)

//go:embed yuv_test_cases.json
var yuvTestCasesJSON []byte

func TestYUV_ColorToYUVBatch(t *testing.T) {
	type testcase struct {
		Name  string `json:"name"`
		Input struct {
			RGB    []uint8 `json:"rgb"`
			Width  int     `json:"width"`
			Height int     `json:"height"`
		} `json:"input"`
		Expected struct {
			YUV []float32 `json:"yuv"`
		} `json:"expected"`
	}
	var test []testcase
	err := json.Unmarshal(yuvTestCasesJSON, &test)
	require.NoError(t, err)

	// abs function for float32
	abs := func(x float32) float32 {
		if x < 0 {
			return -x
		}
		return x
	}

	// robust floating-point comparison function for YUV values
	assertYUVEqual := func(t *testing.T, expected, actual float32, name string, index int) bool {
		const relativeEpsilon = 2e-2    // relative error tolerance (2%)
		const absoluteDelta = 1.0       // absolute error tolerance (1.0 for YUV range)
		const smallValueThreshold = 5.0 // threshold for small values

		// for small values, use absolute error comparison only
		if abs(expected) < smallValueThreshold && abs(actual) < smallValueThreshold {
			return assert.InDelta(t, expected, actual, absoluteDelta, "%s[%d] (small values) expected=%f, got=%f", name, index, expected, actual)
		}

		// otherwise use relative error comparison
		return assert.InEpsilon(t, expected, actual, relativeEpsilon, "%s[%d] expected=%f, got=%f", name, index, expected, actual)
	}

	for _, tt := range test {
		t.Run(tt.Name, func(t *testing.T) {
			// Convert RGB data to color.Color slice
			rgbData := tt.Input.RGB
			require.Equal(t, 0, len(rgbData)%3, "RGB data length must be multiple of 3")

			pixelCount := len(rgbData) / 3
			pixels := make([]color.Color, pixelCount)

			for i := 0; i < pixelCount; i++ {
				r := rgbData[i*3]
				g := rgbData[i*3+1]
				b := rgbData[i*3+2]
				pixels[i] = color.RGBA{R: r, G: g, B: b, A: 255}
			}

			// Prepare output slices
			y := make([]float32, pixelCount)
			u := make([]float32, pixelCount)
			v := make([]float32, pixelCount)
			alpha := make([]uint16, pixelCount)

			// Execute YUV conversion
			yuv.ColorToYUVBatch(pixels, y, u, v, alpha)

			// Verify results
			expectedYUV := tt.Expected.YUV
			require.Equal(t, len(expectedYUV), pixelCount*3, "Expected YUV data length mismatch")

			for i := 0; i < pixelCount; i++ {
				expectedY := expectedYUV[i*3]
				expectedU := expectedYUV[i*3+1]
				expectedV := expectedYUV[i*3+2]

				assertYUVEqual(t, expectedY, y[i], "Y", i)
				assertYUVEqual(t, expectedU, u[i], "U", i)
				assertYUVEqual(t, expectedV, v[i], "V", i)

				// Alpha should always be 65535 (255 * 257)
				assert.Equal(t, uint16(65535), alpha[i], "Alpha[%d]", i)
			}
		})
	}
}

func TestYUV_YUVToRGBA64Batch(t *testing.T) {
	type testcase struct {
		Name  string `json:"name"`
		Input struct {
			RGB    []uint8 `json:"rgb"`
			Width  int     `json:"width"`
			Height int     `json:"height"`
		} `json:"input"`
		Expected struct {
			YUV []float32 `json:"yuv"`
		} `json:"expected"`
	}
	var test []testcase
	err := json.Unmarshal(yuvTestCasesJSON, &test)
	require.NoError(t, err)

	for _, tt := range test {
		t.Run(tt.Name, func(t *testing.T) {
			// Use expected YUV values as input for reverse conversion
			expectedYUV := tt.Expected.YUV
			pixelCount := len(expectedYUV) / 3

			// Prepare YUV input slices
			y := make([]float32, pixelCount)
			u := make([]float32, pixelCount)
			v := make([]float32, pixelCount)
			alpha := make([]uint16, pixelCount)

			for i := 0; i < pixelCount; i++ {
				y[i] = expectedYUV[i*3]
				u[i] = expectedYUV[i*3+1]
				v[i] = expectedYUV[i*3+2]
				alpha[i] = uint16(65535) // Full alpha
			}

			// Execute YUV to RGBA conversion
			pixels := make([]color.RGBA64, pixelCount)
			yuv.YUVToRGBA64Batch(y, u, v, alpha, pixels)

			// Verify results (should approximately match original RGB)
			originalRGB := tt.Input.RGB

			for i := 0; i < pixelCount; i++ {
				expectedR := uint16(originalRGB[i*3]) * 257 // Convert 8-bit to 16-bit
				expectedG := uint16(originalRGB[i*3+1]) * 257
				expectedB := uint16(originalRGB[i*3+2]) * 257

				// Allow some tolerance for round-trip conversion
				const tolerance = uint16(512) // ~2 in 8-bit space

				assert.InDelta(t, expectedR, pixels[i].R, float64(tolerance), "R[%d] expected=%d, got=%d", i, expectedR, pixels[i].R)
				assert.InDelta(t, expectedG, pixels[i].G, float64(tolerance), "G[%d] expected=%d, got=%d", i, expectedG, pixels[i].G)
				assert.InDelta(t, expectedB, pixels[i].B, float64(tolerance), "B[%d] expected=%d, got=%d", i, expectedB, pixels[i].B)
				assert.Equal(t, uint16(65535), pixels[i].A, "A[%d]", i)
			}
		})
	}
}

func TestYUV_RoundTrip(t *testing.T) {
	// Test round-trip conversion: RGB -> YUV -> RGB
	testPixels := []color.RGBA{
		{R: 255, G: 0, B: 0, A: 255},     // Red
		{R: 0, G: 255, B: 0, A: 255},     // Green
		{R: 0, G: 0, B: 255, A: 255},     // Blue
		{R: 255, G: 255, B: 255, A: 255}, // White
		{R: 0, G: 0, B: 0, A: 255},       // Black
		{R: 128, G: 128, B: 128, A: 255}, // Gray
		{R: 100, G: 150, B: 200, A: 255}, // Custom color
	}

	pixels := make([]color.Color, len(testPixels))
	for i, p := range testPixels {
		pixels[i] = p
	}

	// Convert to YUV
	y := make([]float32, len(pixels))
	u := make([]float32, len(pixels))
	v := make([]float32, len(pixels))
	alpha := make([]uint16, len(pixels))

	yuv.ColorToYUVBatch(pixels, y, u, v, alpha)

	// Convert back to RGB
	resultPixels := make([]color.RGBA64, len(pixels))
	yuv.YUVToRGBA64Batch(y, u, v, alpha, resultPixels)

	// Verify round-trip accuracy
	const tolerance = uint16(512) // Allow ~2 in 8-bit space for round-trip error

	for i, original := range testPixels {
		expectedR := uint16(original.R) * 257
		expectedG := uint16(original.G) * 257
		expectedB := uint16(original.B) * 257

		t.Logf("Pixel %d: Original RGB(%d,%d,%d) -> Result RGB16(%d,%d,%d)",
			i, original.R, original.G, original.B,
			resultPixels[i].R, resultPixels[i].G, resultPixels[i].B)

		assert.InDelta(t, expectedR, resultPixels[i].R, float64(tolerance),
			"R[%d] round-trip error too large", i)
		assert.InDelta(t, expectedG, resultPixels[i].G, float64(tolerance),
			"G[%d] round-trip error too large", i)
		assert.InDelta(t, expectedB, resultPixels[i].B, float64(tolerance),
			"B[%d] round-trip error too large", i)
		assert.Equal(t, uint16(65535), resultPixels[i].A, "A[%d]", i)
	}
}
