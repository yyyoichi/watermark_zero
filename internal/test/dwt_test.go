package test

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yyyoichi/watermark_zero/internal/dwt"
)

//go:embed testcase/dwt_test_cases.json
var dwtTestCasesJSON []byte

func TestDWT_HaarDWT(t *testing.T) {
	type testcase struct {
		Name  string `json:"name"`
		Input struct {
			Data   []float32 `json:"data"`
			Width  int       `json:"width"`
			Height int       `json:"height"`
		} `json:"input"`
		Expected struct {
			CA []float32 `json:"cA"`
			CH []float32 `json:"cH"`
			CV []float32 `json:"cV"`
			CD []float32 `json:"cD"`
		} `json:"expected"`
	}
	var test []testcase
	err := json.Unmarshal(dwtTestCasesJSON, &test)
	require.NoError(t, err)

	// abs function for float32
	abs := func(x float32) float32 {
		if x < 0 {
			return -x
		}
		return x
	}

	// robust floating-point comparison function
	assertFloatEqual := func(t *testing.T, expected, actual float32, name string, index int) bool {
		const relativeEpsilon = 1e-2     // relative error tolerance (1%)
		const absoluteDelta = 1e-5       // absolute error tolerance
		const smallValueThreshold = 1e-4 // threshold for small values

		// for very small values, use absolute error comparison only
		if abs(expected) < smallValueThreshold && abs(actual) < smallValueThreshold {
			return assert.InDelta(t, expected, actual, absoluteDelta, "%s[%d] (small values) expected=%f, got=%f", name, index, expected, actual)
		}

		// otherwise use relative error comparison
		return assert.InEpsilon(t, expected, actual, relativeEpsilon, "%s[%d] expected=%f, got=%f", name, index, expected, actual)
	}

	for _, tt := range test {
		t.Run(tt.Name, func(t *testing.T) {
			m := make([]int, len(tt.Input.Data))
			for i := range m {
				m[i] = i
			}
			result := dwt.HaarDWT(tt.Input.Data, tt.Input.Width, m)

			require.Equal(t, len(tt.Expected.CA), len(result[0]), "cA length mismatch")
			for i := range tt.Expected.CA {
				assertFloatEqual(t, tt.Expected.CA[i], result[0][i], "cA", i)
			}

			require.Equal(t, len(tt.Expected.CH), len(result[1]), "cH length mismatch")
			for i := range tt.Expected.CH {
				assertFloatEqual(t, tt.Expected.CH[i], result[1][i], "cH", i)
			}

			require.Equal(t, len(tt.Expected.CV), len(result[2]), "cV length mismatch")
			for i := range tt.Expected.CV {
				assertFloatEqual(t, tt.Expected.CV[i], result[2][i], "cV", i)
			}

			require.Equal(t, len(tt.Expected.CD), len(result[3]), "cD length mismatch")
			for i := range tt.Expected.CD {
				assertFloatEqual(t, tt.Expected.CD[i], result[3][i], "cD", i)
			}

			data := dwt.HaarIDWT(result, tt.Input.Width, tt.Input.Height, m)
			require.Equal(t, len(tt.Input.Data), len(data), "Reconstructed data length mismatch")
			for i := range tt.Input.Data {
				assertFloatEqual(t, tt.Input.Data[i], data[i], "Reconstructed data", i)
			}
		})
		t.Run(fmt.Sprintf("%s/WaveletGet", tt.Name), func(t *testing.T) {
			for _, b := range [][2]int{
				{2, 2},
				{2, 4},
				{4, 4},
				{4, 8},
				{8, 8},
			} {
				indexMap := dwt.NewBlockMap(tt.Input.Width/2, tt.Input.Height/2, b[0], b[1]).GetMap()
				want := dwt.HaarDWT(tt.Input.Data, tt.Input.Width, indexMap)
				wavelets := dwt.New(tt.Input.Data, tt.Input.Width)
				got := wavelets.Get(indexMap)
				require.Equal(t, want, got, "Wavelets mismatch ", b[0], b[1])
			}
		})
	}
}

func TestDWT_BlockMap(t *testing.T) {
	tests := []struct {
		name                                   string
		width, height, blockWidth, blockHeight int
		expected                               []int
	}{
		{
			name:  "11x11_2x2",
			width: 11, height: 11, blockWidth: 2, blockHeight: 2,
			expected: []int{
				0, 1, 4, 5, 8, 9, 12, 13, 16, 17, 100,
				2, 3, 6, 7, 10, 11, 14, 15, 18, 19, 101,
				20, 21, 24, 25, 28, 29, 32, 33, 36, 37, 102,
				22, 23, 26, 27, 30, 31, 34, 35, 38, 39, 103,
				40, 41, 44, 45, 48, 49, 52, 53, 56, 57, 104,
				42, 43, 46, 47, 50, 51, 54, 55, 58, 59, 105,
				60, 61, 64, 65, 68, 69, 72, 73, 76, 77, 106,
				62, 63, 66, 67, 70, 71, 74, 75, 78, 79, 107,
				80, 81, 84, 85, 88, 89, 92, 93, 96, 97, 108,
				82, 83, 86, 87, 90, 91, 94, 95, 98, 99, 109,
				110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120,
			},
		},
		{
			name:  "8x8_2x2",
			width: 8, height: 8, blockWidth: 2, blockHeight: 2,
			expected: []int{
				0, 1, 4, 5, 8, 9, 12, 13,
				2, 3, 6, 7, 10, 11, 14, 15,
				16, 17, 20, 21, 24, 25, 28, 29,
				18, 19, 22, 23, 26, 27, 30, 31,
				32, 33, 36, 37, 40, 41, 44, 45,
				34, 35, 38, 39, 42, 43, 46, 47,
				48, 49, 52, 53, 56, 57, 60, 61,
				50, 51, 54, 55, 58, 59, 62, 63,
			},
		},
		{
			name:  "12x10_3x3",
			width: 12, height: 10, blockWidth: 3, blockHeight: 3,
			expected: []int{
				0, 1, 2, 9, 10, 11, 18, 19, 20, 27, 28, 29,
				3, 4, 5, 12, 13, 14, 21, 22, 23, 30, 31, 32,
				6, 7, 8, 15, 16, 17, 24, 25, 26, 33, 34, 35,
				36, 37, 38, 45, 46, 47, 54, 55, 56, 63, 64, 65,
				39, 40, 41, 48, 49, 50, 57, 58, 59, 66, 67, 68,
				42, 43, 44, 51, 52, 53, 60, 61, 62, 69, 70, 71,
				72, 73, 74, 81, 82, 83, 90, 91, 92, 99, 100, 101,
				75, 76, 77, 84, 85, 86, 93, 94, 95, 102, 103, 104,
				78, 79, 80, 87, 88, 89, 96, 97, 98, 105, 106, 107,
				108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119,
			},
		},
		{
			name:  "7x6_5x4",
			width: 14, height: 10, blockWidth: 5, blockHeight: 4,
			expected: []int{
				0, 1, 2, 3, 4 /**/, 20, 21, 22, 23, 24, 80, 81, 82, 83,
				5, 6, 7, 8, 9 /**/, 25, 26, 27, 28, 29, 84, 85, 86, 87,
				10, 11, 12, 13, 14, 30, 31, 32, 33, 34, 88, 89, 90, 91,
				15, 16, 17, 18, 19, 35, 36, 37, 38, 39, 92, 93, 94, 95,

				40, 41, 42, 43, 44, 60, 61, 62, 63, 64, 96, 97, 98, 99,
				45, 46, 47, 48, 49, 65, 66, 67, 68, 69, 100, 101, 102, 103,
				50, 51, 52, 53, 54, 70, 71, 72, 73, 74, 104, 105, 106, 107,
				55, 56, 57, 58, 59, 75, 76, 77, 78, 79, 108, 109, 110, 111,

				112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125,
				126, 127, 128, 129, 130, 131, 132, 133, 134, 135, 136, 137, 138, 139,
			},
		},
		{
			name:  "4x4_4x4",
			width: 4, height: 4, blockWidth: 4, blockHeight: 4,
			expected: []int{
				0, 1, 2, 3,
				4, 5, 6, 7,
				8, 9, 10, 11,
				12, 13, 14, 15,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := dwt.NewBlockMap(tt.width, tt.height, tt.blockWidth, tt.blockHeight)
			got := m.GetMap()
			assert.Equal(t, tt.expected, got)
		})
	}
}
