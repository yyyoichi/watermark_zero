package watermark

type BlockShape [2]int

func NewBlockShape(width, height int) BlockShape {
	if width%2 != 0 {
		width += 1
	}
	if height%2 != 0 {
		height += 1
	}
	if width < 4 {
		width = 4
	}
	if height < 4 {
		height = 4
	}
	return [2]int{width / 2, height / 2}
}

func (s BlockShape) blockArea() int {
	return s[0] * s[1]
}

func (s BlockShape) TotalBlocks(i ImageSource) int {
	return s.totalBlocks(i.waveWidth, i.waveHeight)
}

func (s BlockShape) totalBlocks(width, height int) int {
	return (width / s[0]) * (height / s[1])
}

func (s BlockShape) IsZero() bool {
	return len(s) == 0 || s[0] < 2 || s[1] < 2
}

func (s BlockShape) width() int {
	return s[0]
}

func (s BlockShape) height() int {
	return s[1]
}
