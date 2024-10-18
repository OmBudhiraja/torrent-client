package bitfield

// A Bitfield represents the pieces that a peer has
type Bitfield []byte

func New(totalPices int) Bitfield {

	// calculate the number of bytes needed to represent the bitfield
	byteLen := totalPices / 8
	if totalPices%8 != 0 {
		byteLen++
	}

	return make([]byte, byteLen)
}

// HasPiece tells if a bitfield has a particular index set
func (bf Bitfield) HasPiece(index int) bool {
	byteIndex := index / 8
	offset := index % 8
	if byteIndex < 0 || byteIndex >= len(bf) {
		return false
	}
	return bf[byteIndex]>>uint(7-offset)&1 != 0
}

// SetPiece sets a bit in the bitfield
func (bf Bitfield) SetPiece(index int) {
	byteIndex := index / 8
	offset := index % 8

	// silently discard invalid bounded index
	if byteIndex < 0 || byteIndex >= len(bf) {
		return
	}
	bf[byteIndex] |= 1 << uint(7-offset)
}
