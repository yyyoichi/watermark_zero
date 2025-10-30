package dct

import "math"

type DCT struct {
	w, h  int
	phi2d []float64
}

func New(w, h int) *DCT {
	dct := &DCT{w: w, h: h}

	wf := float64(w)
	hf := float64(h)

	// Create 1D basis functions for width (horizontal)
	phiW := make([]float64, w*w)
	for j := range w {
		// i = 0
		phiW[j] = 1.0 / math.Sqrt(wf)
	}
	for i := 1; i < w; i++ {
		for j := range w {
			phiW[i*w+j] = math.Sqrt(2.0/wf) *
				math.Cos(
					(float64(i)*math.Pi*(float64(j)*2+1))/
						(2.0*wf),
				)
		}
	}

	// Create 1D basis functions for height (vertical)
	phiH := make([]float64, h*h)
	for j := range h {
		// i = 0
		phiH[j] = 1.0 / math.Sqrt(hf)
	}
	for i := 1; i < h; i++ {
		for j := range h {
			phiH[i*h+j] = math.Sqrt(2.0/hf) *
				math.Cos(
					(float64(i)*math.Pi*(float64(j)*2+1))/
						(2.0*hf),
				)
		}
	}

	// Create 2D basis functions
	dct.phi2d = make([]float64, w*h*w*h)
	for i := range h { // DCT coefficient row
		for j := range w { // DCT coefficient column
			for x := range h { // input data row
				for y := range w { // input data column
					idx := i*w*w*h + j*w*h + x*w + y
					dct.phi2d[idx] = phiH[i*h+x] * phiW[j*w+y]
				}
			}
		}
	}

	return dct
}

func (dct *DCT) Exec(data []float32) ([]float64, func()) {
	w := dct.w
	h := dct.h
	phi := dct.phi2d
	result := make([]float64, w*h)

	// Forward DCT
	for i := range h { // DCT coefficient row
		for j := range w { // DCT coefficient column
			sum := 0.0
			for x := range h { // input data row
				for y := range w { // input data column
					phiIdx := i*w*w*h + j*w*h + x*w + y
					dataIdx := x*w + y
					sum += phi[phiIdx] * float64(data[dataIdx])
				}
			}
			result[i*w+j] = sum
		}
	}

	// Return inverse DCT function
	idct := func() {
		for i := range h { // output data row
			for j := range w { // output data column
				sum := 0.0
				for x := range h { // DCT coefficient row
					for y := range w { // DCT coefficient column
						phiIdx := x*w*w*h + y*w*h + i*w + j
						dataIdx := x*w + y
						sum += phi[phiIdx] * result[dataIdx]
					}
				}
				data[i*w+j] = float32(sum)
			}
		}
	}
	return result, idct
}
