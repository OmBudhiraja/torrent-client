package p2p

import (
	"fmt"
)

type pieceWork struct {
	index  int
	length int
	hash   [20]byte
}

type pieceResult struct {
	index  int
	length int
	data   []byte
}

func (p *pieceResult) writeToFiles(files []*outputFile, pieceLength int) error {

	pieceOffsetStart := p.index * pieceLength
	pieceOffsetEnd := pieceOffsetStart + p.length

	bytesWritten := 0

	for _, file := range files {
		fileStart := max(pieceOffsetStart, file.startRange)
		fileEnd := min(pieceOffsetEnd, file.endRange)
		writeoffset := fileStart - file.startRange

		// Write the piece data to the file
		n, err := file.file.WriteAt(p.data[bytesWritten:bytesWritten+(fileEnd-fileStart)], int64(writeoffset))
		if err != nil {
			return err
		}

		bytesWritten += n
		file.remainingBytes -= n

	}

	if bytesWritten != p.length {
		return fmt.Errorf("failed to write data to files, piece index: %d, piece length: %d, bytes written: %d", p.index, p.length, bytesWritten)
	}

	return nil
}
