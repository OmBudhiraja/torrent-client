package p2p

import (
	"fmt"
)

type PieceWork struct {
	Index  int
	Length int
	Hash   [20]byte
}

type PieceResult struct {
	Index  int
	Length int
	Data   []byte
}

func (p *PieceResult) WriteToFiles(files []*OutputFile, pieceLength int) error {

	pieceOffsetStart := p.Index * pieceLength
	pieceOffsetEnd := pieceOffsetStart + p.Length

	bytesWritten := 0

	for _, file := range files {
		fileStart := max(pieceOffsetStart, file.startRange)
		fileEnd := min(pieceOffsetEnd, file.endRange)
		writeoffset := fileStart - file.startRange

		// Write the piece data to the file
		n, err := file.file.WriteAt(p.Data[bytesWritten:bytesWritten+(fileEnd-fileStart)], int64(writeoffset))
		if err != nil {
			return err
		}

		bytesWritten += n

	}

	if bytesWritten != p.Length {
		return fmt.Errorf("failed to write data to files, piece index: %d, piece length: %d, bytes written: %d", p.Index, p.Length, bytesWritten)
	}

	return nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
