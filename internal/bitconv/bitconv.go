package bitconv

func BytesToBools(b []byte) []bool {
	bits := make([]bool, 0, len(b)*8)
	for _, bb := range b {
		for i := 7; i >= 0; i-- {
			bits = append(bits, ((bb>>uint(i))&1) == 1)
		}
	}
	return bits
}

func BoolsToBytes(bits []bool) []byte {
	// calculate padded length without modifying input
	n := len(bits)
	paddedLen := n
	if n%8 != 0 {
		paddedLen += 8 - (n % 8)
	}

	// create padded copy
	paddedBits := make([]bool, paddedLen)
	copy(paddedBits, bits)
	// trailing bits are already false (zero value)

	// convert to bytes
	out := make([]byte, paddedLen/8)
	for i := 0; i < len(out); i++ {
		var v byte
		for j := 0; j < 8; j++ {
			if paddedBits[i*8+j] {
				v |= 1 << uint(7-j)
			}
		}
		out[i] = v
	}
	return out
}
