package p2p

import (
	"os"
	"path/filepath"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
	"github.com/codecrafters-io/bittorrent-starter-go/pkg/progressbar"
)

const (
	maxBlockSize = 16384
	maxBacklog   = 5
)

type Torrent struct {
	Name        string
	Peers       []peer.Peer
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	PeerId      []byte
	Files       []File
}

type File struct {
	Length int
	Path   string
}

type outputFile struct {
	path           string
	length         int
	startRange     int
	endRange       int
	remainingBytes int
	file           *os.File
}

func (t *Torrent) Download(outpath string) error {
	workQueue := make(chan *pieceWork, len(t.PieceHashes))
	results := make(chan *pieceResult)

	isMultifile := len(t.Files) > 0

	var outfiles []*outputFile

	progressbar := progressbar.New(len(t.PieceHashes))

	progressbar.Start()
	defer progressbar.Finish()

	if isMultifile {
		outfiles = make([]*outputFile, len(t.Files))
		for index, file := range t.Files {
			path := filepath.Join(outpath, t.Name, file.Path)
			err := os.MkdirAll(filepath.Dir(path), 0755)

			if err != nil {
				return err
			}

			outfile, err := os.Create(path)

			if err != nil {
				return err
			}

			startRange := 0

			if index > 0 {
				startRange = outfiles[index-1].endRange
			}

			outfiles[index] = &outputFile{
				path:           path,
				length:         file.Length,
				file:           outfile,
				startRange:     startRange,
				remainingBytes: file.Length,
				endRange:       startRange + file.Length,
			}

			defer outfile.Close()

		}

	} else {
		outfiles = make([]*outputFile, 1)
		// check if outpath directory exists
		if _, err := os.Stat(outpath); os.IsNotExist(err) {
			err := os.MkdirAll(outpath, 0755)

			if err != nil {
				return err
			}
		}

		filePath := filepath.Join(outpath, t.Name)
		outfile, err := os.Create(filePath)

		if err != nil {
			return err
		}

		outfiles[0] = &outputFile{
			file: outfile,
		}

		defer outfile.Close()
	}

	// store a map of each piece index to the files that it belongs to
	pieceToFileMap := make(map[int][]*outputFile)
	lastFileIndex := 0

	for i, pieceHash := range t.PieceHashes {
		pieceLength := t.getPieceLength(i)

		pieceStartOffset := i * t.PieceLength
		pieceEndOffset := pieceStartOffset + pieceLength

		for lastFileIndex < len(outfiles) {
			file := outfiles[lastFileIndex]
			fileStartOffset := file.startRange
			fileEndOffset := file.endRange

			if pieceEndOffset > fileStartOffset && pieceStartOffset < fileEndOffset {
				pieceToFileMap[i] = append(pieceToFileMap[i], file)
			}

			if pieceEndOffset == fileEndOffset {
				lastFileIndex++
				break
			} else if pieceEndOffset > fileEndOffset {
				lastFileIndex++
			} else {
				break
			}
		}

		workQueue <- &pieceWork{
			index:  i,
			length: pieceLength,
			hash:   pieceHash,
		}

	}

	for _, peer := range t.Peers {
		go t.startWorker(peer, workQueue, results)
	}

	var piecesDownloaded int

	for piecesDownloaded < len(t.PieceHashes) {
		piece := <-results

		piecesDownloaded++

		if !isMultifile {
			offset := piece.index * t.PieceLength
			_, err := outfiles[0].file.WriteAt(piece.data, int64(offset))

			if err != nil {
				return err
			}

		} else {

			err := piece.writeToFiles(pieceToFileMap[piece.index], t.PieceLength)

			if err != nil {
				return err
			}
		}

		progressbar.Update(piecesDownloaded)
	}

	close(workQueue)

	return nil
}

func (t *Torrent) getPieceLength(pieceIndex int) int {

	length := t.PieceLength

	if pieceIndex == len(t.PieceHashes)-1 && t.Length%t.PieceLength != 0 {
		length = t.Length % t.PieceLength
	}

	return length
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Helper function to get the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
