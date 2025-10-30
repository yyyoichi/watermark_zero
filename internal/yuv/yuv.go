package yuv

import "image/color"

// https://github.com/opencv/opencv/blob/0e88b49a53842f0f7cdc4c61b98c283be7e5057c/modules/imgproc/src/opencl/color_yuv.cl#L148-L234

const delta = .5
const (
	yr = 0.299
	yg = 0.587
	yb = 0.114
	uf = 0.492
	vf = 0.877
)

func ColorToYUVBatch(pixels []color.Color, y, u, v []float32, alpha []uint16) {

	for i, pixel := range pixels {
		r32, g32, b32, a32 := pixel.RGBA()
		r := float32(r32 >> 8)
		g := float32(g32 >> 8)
		b := float32(b32 >> 8)

		yVal := yr*r + yg*g + yb*b
		y[i] = yVal
		u[i] = uf*(b-yVal) + delta
		v[i] = vf*(r-yVal) + delta
		alpha[i] = uint16(a32)
	}
}

const (
	vr = 1.140
	ug = -0.395
	vg = -0.581
	ub = 2.032
)

func YUVToRGBA64Batch(y, u, v []float32, alpha []uint16, pixels []color.RGBA64) {

	for i := range pixels {
		yVal := y[i]
		uDelta := u[i] - delta
		vDelta := v[i] - delta

		r := yVal + vr*vDelta
		g := yVal + ug*uDelta + vg*vDelta
		b := yVal + ub*uDelta

		pixels[i] = color.RGBA64{
			R: clip16(r),
			G: clip16(g),
			B: clip16(b),
			A: alpha[i],
		}
	}
}

func clip16(rgb float32) uint16 {
	if rgb < 0 {
		return 0
	}
	if rgb > 255 {
		return 65535
	}
	return uint16(rgb * 257.0)
}
