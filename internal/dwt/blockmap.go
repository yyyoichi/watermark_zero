package dwt

type BlockMap struct {
	width, height           int // Original image dimensions
	blockWidth, blockHeight int // Individual block dimensions

	allocWidth, allocHeight int // Total allocated dimensions for block proceming
	marginWidth             int // Width of unallocated left margin area
	blockArea               int // Area of a single block (blockWidth * blockHeight)
	totalAllocArea          int // Total allocated area (allocWidth * allocHeight = nbx * nby * blockArea)
	blockRowArea            int // Area of one block row (allocWidth * blockHeight, excluding margin)
}

func NewBlockMap(w, h, bw, bh int) BlockMap {
	var m = BlockMap{
		width:       w,
		height:      h,
		blockWidth:  bw,
		blockHeight: bh,
	}
	countBlockX, countBlockY := w/bw, h/bh
	m.allocWidth, m.allocHeight = countBlockX*bw, countBlockY*bh
	m.marginWidth = w - m.allocWidth
	m.blockArea = m.blockWidth * m.blockHeight
	m.totalAllocArea = m.allocWidth * m.allocHeight
	m.blockRowArea = m.allocWidth * m.blockHeight
	return m
}

func (m BlockMap) GetMap() []int {
	result := make([]int, m.width*m.height)
	for i := range result {
		result[i] = m.get(i)
	}
	return result
}

func (m BlockMap) get(i int) int {
	x, y := i%m.width, i/m.width
	if m.allocHeight <= y {
		// bottom margin
		return i
	}
	if mx := x - m.allocWidth; mx >= 0 {
		// left margin
		return m.totalAllocArea +
			y*m.marginWidth + mx
	}
	// in block
	brow, bcol := y/m.blockHeight, x/m.blockWidth
	start := brow*m.blockRowArea + bcol*m.blockArea
	bx, by := x%m.blockWidth, y%m.blockHeight
	return start + by*m.blockWidth + bx
}
