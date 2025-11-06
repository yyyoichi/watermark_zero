package dwt

import (
	"math"
)

func HaarDWT(data []float32, w int, indexMap []int) [][]float32 {
	h := len(data) / w

	hw, hh := (w+1)/2, (h+1)/2
	l := hw * hh
	cA := make([]float32, l)
	cH := make([]float32, l)
	cV := make([]float32, l)
	cD := make([]float32, l)

	if indexMap == nil || len(indexMap) != l {
		indexMap = make([]int, l)
		for i := range l {
			indexMap[i] = i
		}
	}

	for y0 := 0; y0 < h; y0 += 2 {
		var y1 int
		if y0+1 < h {
			y1 = y0 + 1
		} else {
			y1 = y0
		}
		for x0 := 0; x0 < w; x0 += 2 {
			var x1 int
			if x0+1 < w {
				x1 = x0 + 1
			} else {
				x1 = x0
			}
			a1, d1 := cacd(data[y0*w+x0], data[y1*w+x0])
			a2, d2 := cacd(data[y0*w+x1], data[y1*w+x1])

			idx := indexMap[(y0/2)*hw+(x0/2)]
			cA[idx], cV[idx] = cacd(a1, a2)
			cH[idx], cD[idx] = cacd(d1, d2)
		}
	}

	return [][]float32{cA, cH, cV, cD}
}

func HaarIDWT(result [][]float32, w, h int, indexMap []int) []float32 {
	data := make([]float32, w*h)
	var (
		cA = result[0]
		cH = result[1]
		cV = result[2]
		cD = result[3]
	)
	hw := (w + 1) / 2
	for y0 := 0; y0 < h; y0 += 2 {
		for x0 := 0; x0 < w; x0 += 2 {
			idx := indexMap[(y0/2)*hw+(x0/2)]

			a1, a2 := icacd(cA[idx], cV[idx])
			d1, d2 := icacd(cH[idx], cD[idx])

			v1, v2 := icacd(a1, d1)
			v3, v4 := icacd(a2, d2)

			data[y0*w+x0] = v1
			if y0+1 < h {
				data[(y0+1)*w+x0] = v2
			}
			if x0+1 < w {
				data[y0*w+(x0+1)] = v3
			}
			if y0+1 < h && x0+1 < w {
				data[(y0+1)*w+(x0+1)] = v4
			}
		}
	}
	return data
}

func cacd(v1, v2 float32) (float32, float32) {
	avr := (v1 + v2) / 2.0
	return avr * math.Sqrt2, (v1 - avr) * math.Sqrt2
}
func icacd(a, d float32) (float32, float32) {
	avr := a / math.Sqrt2
	return avr + d/math.Sqrt2, avr - d/math.Sqrt2
}

type Wavelets struct {
	hw, hh   int
	original [][]float32
}

func New(data []float32, w int) *Wavelets {
	h := len(data) / w
	wavelets := Wavelets{
		hw: (w + 1) / 2,
		hh: (h + 1) / 2,
	}
	wavelets.original = HaarDWT(data, w, nil)
	return &wavelets
}

func (w *Wavelets) Get(blockW, blockH int) [][]float32 {
	l := w.hw * w.hh
	result := [][]float32{
		make([]float32, l),
		make([]float32, l),
		make([]float32, l),
		make([]float32, l),
	}
	indexMap := NewBlockMap(w.hw, w.hh, blockW, blockH).GetMap()
	for j, o := range w.original {
		for i, v := range o {
			idx := indexMap[i]
			result[j][idx] = v
		}
	}
	return result
}
