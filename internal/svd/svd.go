package svd

import (
	"fmt"

	"gonum.org/v1/gonum/mat"
)

type SVD struct {
	w, h int
}

func New(w, h int) *SVD {
	return &SVD{w: w, h: h}
}

func (svd *SVD) Exec(data []float64) (s []float64, isvd func(), err error) {
	w := svd.w
	h := svd.h

	// Treat as rectangular matrix (h rows x w columns)
	a := mat.NewDense(h, w, data)
	var result mat.SVD
	if ok := result.Factorize(a, mat.SVDFull); !ok {
		return nil, nil, fmt.Errorf("cannot factorize")
	}

	s = result.Values(nil)
	isvd = func() {
		// Number of singular values is min(h, w)
		minDim := min(h, w)

		// Create singular value matrix (h x w)
		sigma := mat.NewDense(h, w, nil)
		for i := 0; i < minDim && i < len(s); i++ {
			sigma.Set(i, i, s[i])
		}

		// Get U and V matrices
		var u, v mat.Dense
		result.UTo(&u)
		result.VTo(&v)

		// Reconstruct A = U * Σ * V^T
		var res mat.Dense
		res.Product(&u, sigma, v.T())

		// Copy result back to data
		resData := res.RawMatrix().Data
		if len(resData) != len(data) {
			// Safe copy to avoid size mismatch errors
			copy(data, resData[:min(len(data), len(resData))])
		} else {
			copy(data, resData)
		}
	}
	return
}

// min function for integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
