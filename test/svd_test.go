package test

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyyoichi/watermark_zero/internal/svd"
)

//go:embed svd_test_cases.json
var svdTestCasesJSON []byte

func TestSVD_Exec(t *testing.T) {
	type testcase struct {
		Name  string `json:"name"`
		Input struct {
			Data   []float64 `json:"data"`
			Width  int       `json:"width"`
			Height int       `json:"height"`
		} `json:"input"`
		Expected struct {
			SingularValues []float64 `json:"singular_values"`
			U              []float64 `json:"u"`
			Vt             []float64 `json:"vt"`
		} `json:"expected"`
	}
	var test []testcase
	err := json.Unmarshal(svdTestCasesJSON, &test)
	require.NoError(t, err)

	// abs function for float64
	abs := func(x float64) float64 {
		if x < 0 {
			return -x
		}
		return x
	}

	// robust floating-point comparison function for SVD values
	assertSVDEqual := func(t *testing.T, expected, actual float64, name string, index int) bool {
		const relativeEpsilon = 1e-10     // relative error tolerance (very strict for SVD)
		const absoluteDelta = 1e-12       // absolute error tolerance for very small values
		const smallValueThreshold = 1e-10 // threshold for small values

		// for very small values, use absolute error comparison only
		if abs(expected) < smallValueThreshold && abs(actual) < smallValueThreshold {
			return assert.InDelta(t, expected, actual, absoluteDelta, "%s[%d] (small values) expected=%e, got=%e", name, index, expected, actual)
		}

		// otherwise use relative error comparison
		return assert.InEpsilon(t, expected, actual, relativeEpsilon, "%s[%d] expected=%e, got=%e", name, index, expected, actual)
	}

	for _, tt := range test {
		t.Run(tt.Name, func(t *testing.T) {
			// Create SVD instance
			svdInstance := svd.New(tt.Input.Width, tt.Input.Height)

			// Execute SVD
			singularValues, _, err := svdInstance.Exec(tt.Input.Data)
			require.NoError(t, err)

			// Verify singular values (this is the most important part for SVD)
			expectedS := tt.Expected.SingularValues
			require.Equal(t, len(expectedS), len(singularValues), "Singular values length mismatch")

			for i := range expectedS {
				assertSVDEqual(t, expectedS[i], singularValues[i], "SingularValue", i)
			}
		})
	}
}

func TestSVD_RoundTrip(t *testing.T) {
	// Test round-trip: original -> SVD -> reconstruct -> should equal original
	testCases := []struct {
		name   string
		width  int
		height int
		data   []float64
	}{
		{
			name:   "2x2_simple",
			width:  2,
			height: 2,
			data:   []float64{3, 1, 1, 3},
		},
		{
			name:   "3x3_identity",
			width:  3,
			height: 3,
			data:   []float64{1, 0, 0, 0, 1, 0, 0, 0, 1},
		},
		{
			name:   "3x2_rectangular",
			width:  2,
			height: 3,
			data:   []float64{1, 2, 3, 4, 5, 6},
		},
		{
			name:   "2x3_rectangular",
			width:  3,
			height: 2,
			data:   []float64{1, 2, 3, 4, 5, 6},
		},
		{
			name:   "3x3_diagonal",
			width:  3,
			height: 3,
			data:   []float64{5, 0, 0, 0, 3, 0, 0, 0, 1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Keep original data for comparison
			original := make([]float64, len(tc.data))
			copy(original, tc.data)

			// Create SVD instance and execute
			svdInstance := svd.New(tc.width, tc.height)
			_, isvd, err := svdInstance.Exec(tc.data)
			require.NoError(t, err)

			// Execute inverse SVD (reconstruction)
			isvd()

			// Verify round-trip accuracy
			const tolerance = 1e-10
			for i, expectedVal := range original {
				// Use InDelta instead of InEpsilon to handle zero values properly
				assert.InDelta(t, expectedVal, tc.data[i], tolerance,
					"Round-trip error at index %d: expected=%e, got=%e", i, expectedVal, tc.data[i])
			}
		})
	}
}

func TestSVD_Properties(t *testing.T) {
	// Test mathematical properties of SVD

	t.Run("singular_values_non_negative", func(t *testing.T) {
		// Singular values should always be non-negative and in descending order
		testData := []float64{4, 2, 1, 3, 5, 6, 7, 8, 9}
		width, height := 3, 3

		svdInstance := svd.New(width, height)
		singularValues, _, err := svdInstance.Exec(testData)
		require.NoError(t, err)

		// Check non-negative
		for i, s := range singularValues {
			assert.GreaterOrEqual(t, s, 0.0, "Singular value[%d] should be non-negative", i)
		}

		// Check descending order
		for i := 1; i < len(singularValues); i++ {
			assert.GreaterOrEqual(t, singularValues[i-1], singularValues[i],
				"Singular values should be in descending order: s[%d]=%e >= s[%d]=%e",
				i-1, singularValues[i-1], i, singularValues[i])
		}
	})

	t.Run("rank_deficient_matrix", func(t *testing.T) {
		// Test with a rank-deficient matrix (rank < min(m,n))
		// This matrix has rank 1 (all rows are multiples of [1, 2, 3])
		testData := []float64{
			1, 2, 3,
			2, 4, 6,
			3, 6, 9,
		}
		width, height := 3, 3

		svdInstance := svd.New(width, height)
		singularValues, _, err := svdInstance.Exec(testData)
		require.NoError(t, err)

		// Should have one large singular value and two near-zero ones
		assert.Greater(t, singularValues[0], 1.0, "First singular value should be large")
		assert.Less(t, singularValues[1], 1e-10, "Second singular value should be near zero")
		assert.Less(t, singularValues[2], 1e-10, "Third singular value should be near zero")
	})

	t.Run("identity_matrix", func(t *testing.T) {
		// Identity matrix should have all singular values equal to 1
		testData := []float64{1, 0, 0, 0, 1, 0, 0, 0, 1}
		width, height := 3, 3

		svdInstance := svd.New(width, height)
		singularValues, _, err := svdInstance.Exec(testData)
		require.NoError(t, err)

		for i, s := range singularValues {
			assert.InDelta(t, 1.0, s, 1e-10, "Identity matrix singular value[%d] should be 1.0", i)
		}
	})
}
