package bench

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/yyyoichi/watermark_zero/internal/dct"
	"github.com/yyyoichi/watermark_zero/internal/dwt"
	"github.com/yyyoichi/watermark_zero/internal/svd"
)

func BenchmarkBlockMap(b *testing.B) {
	waveletBlockWidth, waveletBlockHeight := 4, 4
	d1, d2 := 21, 9
	fd1, fd2 := float64(d1), float64(d2)
	embedFunc := func(s0, s1, bit float64) (r0 float64, r1 float64) {
		r0 = (float64(int(s0)/d1) + 1.0/4.0 + 1.0/2.0*0.5*bit) * fd1
		r1 = (float64(int(s1)/d2) + 1.0/4.0 + 1.0/2.0*0.5*bit) * fd2
		return
	}
	svd := svd.New(waveletBlockWidth, waveletBlockHeight)
	dct := dct.New(waveletBlockWidth, waveletBlockHeight)
	getMarkBit := func(blockAt int) float64 {
		if blockAt%2 == 0 {
			return 1.0
		}
		return 0.0
	}
	genSrc := func(w, h int) []float32 {
		src := make([]float32, w*h)
		for i := range src {
			src[i] = rand.Float32() * 255.0
		}
		return src
	}

	for _, size := range [][2]int{{1280, 720}, {1920, 1080}, {3840, 2160}} {
		srcWidth, srcHeight := size[0], size[1]
		waveletWidth, waveletHeight := srcWidth/2, srcHeight/2
		get := func(src []float32, startX, startY int) []float32 {
			block := make([]float32, waveletBlockWidth*waveletBlockHeight)
			for by := 0; by < waveletBlockHeight; by++ {
				blockI := by * waveletBlockWidth
				srcI := (startY+by)*waveletWidth + startX
				copy(block[blockI:blockI+waveletBlockWidth:blockI+waveletBlockWidth], src[srcI:srcI+waveletBlockWidth:srcI+waveletBlockWidth])
			}
			return block
		}
		set := func(src []float32, startX, startY int, block []float32) {
			for by := 0; by < waveletBlockHeight; by++ {
				blockI := by * waveletBlockWidth
				srcI := (startY+by)*waveletWidth + startX
				copy(src[srcI:srcI+waveletBlockWidth:srcI+waveletBlockWidth], block[blockI:blockI+waveletBlockWidth:blockI+waveletBlockWidth])
			}
		}

		img := genSrc(srcWidth, srcHeight)
		b.Run(fmt.Sprintf("noBlockMap_%dx%d", srcWidth, srcHeight), func(b *testing.B) {
			for b.Loop() {
				wavelets := dwt.HaarDWT(img, srcWidth, nil)
				src := wavelets[0] // cA
				blockAt := 0
				for startY := 0; startY < waveletHeight; startY += waveletBlockHeight {
					for startX := 0; startX < waveletWidth; startX += waveletBlockWidth {
						b := get(src, startX, startY)
						v, idct := dct.Exec(b)
						s, isvd, _ := svd.Exec(v)
						bit := getMarkBit(blockAt)
						s[0], s[1] = embedFunc(s[0], s[1], bit)
						isvd()
						idct()
						set(src, startX, startY, b)
						blockAt++
					}
				}
				dist := dwt.HaarIDWT(wavelets, srcWidth, srcHeight, nil)
				_ = dist
			}
		})

		b.Run(fmt.Sprintf("withBlockMap_%dx%d", srcWidth, srcHeight), func(b *testing.B) {
			for b.Loop() {
				blockMap := dwt.NewBlockMap(waveletWidth, waveletHeight, waveletBlockWidth, waveletBlockHeight).GetMap()
				wavelets := dwt.HaarDWT(img, srcWidth, blockMap)
				src := wavelets[0] // cA
				totalBlocks := (waveletWidth / waveletBlockWidth) * (waveletHeight / waveletBlockHeight)
				for at := range totalBlocks {
					block := src[at*waveletBlockWidth*waveletBlockHeight : (at+1)*waveletBlockWidth*waveletBlockHeight : (at+1)*waveletBlockWidth*waveletBlockHeight]
					v, idct := dct.Exec(block)
					s, isvd, _ := svd.Exec(v)
					bit := getMarkBit(at)
					s[0], s[1] = embedFunc(s[0], s[1], bit)
					isvd()
					idct()
				}
				dist := dwt.HaarIDWT(wavelets, srcWidth, srcHeight, blockMap)
				_ = dist
			}
		})
	}
}

func BenchmarkBlock(b *testing.B) {
	blockWidth, blockHeight := 8, 8
	dosomething := func(block []float32) {
		sum := float32(0)
		for _, v := range block {
			sum += v
		}
		for i := range block {
			block[i] = sum / float32(len(block))
		}
	}
	genSrc := func() []float32 {
		src := make([]float32, 1920*1080)
		for i := range src {
			src[i] = float32(i % 256)
		}
		return src
	}
	get := func(src []float32, startX, startY int) []float32 {
		block := make([]float32, blockWidth*blockHeight)
		for by := 0; by < blockHeight; by++ {
			blockI := by * blockWidth
			srcI := (startY+by)*1920 + startX
			copy(block[blockI:blockI+blockWidth:blockI+blockWidth], src[srcI:srcI+blockWidth:srcI+blockWidth])
		}
		return block
	}
	set := func(src []float32, startX, startY int, block []float32) {
		for by := 0; by < blockHeight; by++ {
			blockI := by * blockWidth
			srcI := (startY+by)*1920 + startX
			copy(src[srcI:srcI+blockWidth:srcI+blockWidth], block[blockI:blockI+blockWidth:blockI+blockWidth])
		}
	}
	refer := func(ordered []float32, blockAt int) []float32 {
		size := blockWidth * blockHeight
		return ordered[blockAt*size : (blockAt+1)*size : (blockAt+1)*size]
	}
	b.Run("Before", func(b *testing.B) {
		for b.Loop() {
			src := genSrc()
			for startY := 0; startY < 1080; startY += blockHeight {
				for startX := 0; startX < 1920; startX += blockWidth {
					b := get(src, startX, startY)
					dosomething(b)
					set(src, startX, startY, b)
				}
			}
			_ = src
		}
	})
	b.Run("After", func(b *testing.B) {
		for b.Loop() {
			src := genSrc()
			blockMap := dwt.NewBlockMap(1920, 1080, blockWidth, blockHeight).GetMap()
			blockMajarSrc := make([]float32, len(src))
			for i, v := range blockMap {
				blockMajarSrc[i] = src[v]
			}
			totalBlocks := (1920 / blockWidth) * (1080 / blockHeight)
			for at := range totalBlocks {
				block := refer(blockMajarSrc, at)
				dosomething(block)
			}
			for i, v := range blockMap {
				src[v] = blockMajarSrc[i]
			}
			_ = src
		}
	})
}
