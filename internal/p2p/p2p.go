package p2p

import (
	"os"
	"path/filepath"

	"github.com/OmBudhiraja/torrent-client/internal/peer"
	"github.com/OmBudhiraja/torrent-client/pkg/progressbar"
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
	Outpath     string
}

type File struct {
	Length int
	Path   string
}

type OutputFile struct {
	length     int
	startRange int
	endRange   int
	file       *os.File
}

type DownloadSessionManger struct {
	WorkQueue      chan *PieceWork
	Results        chan *PieceResult
	Outfiles       []*OutputFile
	PieceToFileMap map[int][]*OutputFile
	T              *Torrent
}

func (t *Torrent) Initiate() (*DownloadSessionManger, error) {

	workQueue := make(chan *PieceWork, len(t.PieceHashes))
	results := make(chan *PieceResult)

	isMultifile := len(t.Files) > 0

	var outfiles []*OutputFile

	if isMultifile {
		outfiles = make([]*OutputFile, len(t.Files))
		for index, file := range t.Files {
			path := filepath.Join(t.Outpath, t.Name, file.Path)
			err := os.MkdirAll(filepath.Dir(path), 0755)

			if err != nil {
				return nil, err
			}

			outfile, err := os.Create(path)

			if err != nil {
				return nil, err
			}

			startRange := 0

			if index > 0 {
				startRange = outfiles[index-1].endRange
			}

			outfiles[index] = &OutputFile{
				length:     file.Length,
				file:       outfile,
				startRange: startRange,
				endRange:   startRange + file.Length,
			}

		}

	} else {
		outfiles = make([]*OutputFile, 1)
		// check if outpath directory exists
		if _, err := os.Stat(t.Outpath); os.IsNotExist(err) {
			err := os.MkdirAll(t.Outpath, 0755)

			if err != nil {
				return nil, err
			}
		}

		filePath := filepath.Join(t.Outpath, t.Name)
		outfile, err := os.Create(filePath)

		if err != nil {
			return nil, err
		}

		outfiles[0] = &OutputFile{
			file:       outfile,
			startRange: 0,
			endRange:   t.Length,
			length:     t.Length,
		}

	}

	// store a map of each piece index to the files that it belongs to
	pieceToFileMap := make(map[int][]*OutputFile)
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

		workQueue <- &PieceWork{
			Index:  i,
			Length: pieceLength,
			Hash:   pieceHash,
		}

	}

	return &DownloadSessionManger{
		WorkQueue:      workQueue,
		Results:        results,
		Outfiles:       outfiles,
		PieceToFileMap: pieceToFileMap,
	}, nil
}

func (t *Torrent) Download() error {

	progressbar := progressbar.New(len(t.PieceHashes))

	progressbar.Start()
	defer progressbar.Finish()

	dsm, err := t.Initiate()

	if err != nil {
		return err
	}

	defer dsm.CloseFiles()

	for _, peer := range t.Peers {
		go t.StartWorker(peer, dsm.WorkQueue, dsm.Results)
	}

	var piecesDownloaded int

	for piecesDownloaded < len(t.PieceHashes) {
		piece := <-dsm.Results
		piecesDownloaded++

		err = piece.WriteToFiles(dsm.PieceToFileMap[piece.Index], t.PieceLength)

		if err != nil {
			return err
		}

		progressbar.Update(piecesDownloaded)
	}

	close(dsm.WorkQueue)

	return nil
}

func (t *Torrent) getPieceLength(pieceIndex int) int {

	length := t.PieceLength

	if pieceIndex == len(t.PieceHashes)-1 && t.Length%t.PieceLength != 0 {
		length = t.Length % t.PieceLength
	}

	return length
}

func (dsm *DownloadSessionManger) CloseFiles() {
	for _, file := range dsm.Outfiles {
		file.file.Close()
	}
}
