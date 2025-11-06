package test

import (
	_ "embed"
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyyoichi/watermark_zero/internal/dct"
)

//go:embed testcase/dct_test_cases.json
var dctTestCasesJSON []byte

func TestDCT_Exec(t *testing.T) {
	type testcase struct {
		Name  string `json:"name"`
		Input struct {
			Data   []float32 `json:"data"`
			Width  int       `json:"width"`
			Height int       `json:"height"`
		} `json:"input"`
		Expected struct {
			DCT []float64 `json:"dct"`
		} `json:"expected"`
	}
	var test []testcase
	err := json.Unmarshal(dctTestCasesJSON, &test)
	require.NoError(t, err)

	// abs function for float64
	abs := func(x float64) float64 {
		if x < 0 {
			return -x
		}
		return x
	}

	// robust floating-point comparison function for DCT values
	assertDCTEqual := func(t *testing.T, expected, actual float64, name string, index int) bool {
		const relativeEpsilon = 1e-4     // relative error tolerance (0.01%)
		const absoluteDelta = 1e-7       // absolute error tolerance for very small values
		const smallValueThreshold = 1e-6 // threshold for small values

		// for very small values, use absolute error comparison only
		if abs(expected) < smallValueThreshold && abs(actual) < smallValueThreshold {
			return assert.InDelta(t, expected, actual, absoluteDelta, "%s[%d] (small values) expected=%e, got=%e", name, index, expected, actual)
		}

		// otherwise use relative error comparison
		return assert.InEpsilon(t, expected, actual, relativeEpsilon, "%s[%d] expected=%e, got=%e", name, index, expected, actual)
	}

	for _, tt := range test {
		t.Run(tt.Name, func(t *testing.T) {
			// Create DCT instance
			dctInstance := dct.New(tt.Input.Width, tt.Input.Height)

			// Execute DCT
			result, _ := dctInstance.Exec(tt.Input.Data)

			// Verify results
			expectedDCT := tt.Expected.DCT
			require.Equal(t, len(expectedDCT), len(result), "DCT result length mismatch")

			for i := range expectedDCT {
				assertDCTEqual(t, expectedDCT[i], result[i], "DCT", i)
			}
		})
	}
}

func TestDCT_RoundTrip(t *testing.T) {
	// Test round-trip: original -> DCT -> IDCT -> should equal original
	testCases := []struct {
		name   string
		width  int
		height int
		data   []float32
	}{
		{
			name:   "2x2_simple",
			width:  2,
			height: 2,
			data:   []float32{1, 2, 3, 4},
		},
		{
			name:   "3x3_sequential",
			width:  3,
			height: 3,
			data:   []float32{1, 2, 3, 4, 5, 6, 7, 8, 9},
		},
		{
			name:   "4x2_rectangular",
			width:  2,
			height: 4,
			data:   []float32{1, 2, 3, 4, 5, 6, 7, 8},
		},
		{
			name:   "2x4_rectangular",
			width:  4,
			height: 2,
			data:   []float32{1, 2, 3, 4, 5, 6, 7, 8},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Keep original data for comparison
			original := make([]float32, len(tc.data))
			copy(original, tc.data)

			// Create DCT instance and execute
			dctInstance := dct.New(tc.width, tc.height)
			_, idct := dctInstance.Exec(tc.data)

			// Execute inverse DCT
			idct()

			// Verify round-trip accuracy
			const tolerance = 1e-5
			for i, expectedVal := range original {
				assert.InEpsilon(t, expectedVal, tc.data[i], tolerance,
					"Round-trip error at index %d: expected=%f, got=%f", i, expectedVal, tc.data[i])
			}
		})
	}
}

func TestDCT_Properties(t *testing.T) {
	// Test mathematical properties of DCT

	t.Run("DC_component", func(t *testing.T) {
		// For constant input, only DC component (0,0) should be non-zero
		width, height := 4, 4
		constantValue := float32(5.0)
		data := make([]float32, width*height)
		for i := range data {
			data[i] = constantValue
		}

		dctInstance := dct.New(width, height)
		result, _ := dctInstance.Exec(data)

		// DC component should be constantValue * sqrt(width * height)
		expectedDC := float64(constantValue) * math.Sqrt(float64(width*height))
		assert.InEpsilon(t, expectedDC, result[0], 1e-5, "DC component mismatch")

		// All other components should be approximately zero
		for i := 1; i < len(result); i++ {
			assert.InDelta(t, 0.0, result[i], 1e-10, "Non-DC component[%d] should be zero", i)
		}
	})

	t.Run("zero_input", func(t *testing.T) {
		// Zero input should produce zero output
		width, height := 3, 3
		data := make([]float32, width*height) // all zeros

		dctInstance := dct.New(width, height)
		result, _ := dctInstance.Exec(data)

		for i, val := range result {
			assert.Equal(t, 0.0, val, "Zero input should produce zero output at index %d", i)
		}
	})
}
