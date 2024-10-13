package p2p

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/client"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/message"
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

	outFilesMap := make(map[int]*outputFile)

	progressbar := progressbar.New(len(t.PieceHashes))

	progressbar.Start()

	defer progressbar.Finish()

	if isMultifile {
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
				startRange = outFilesMap[index-1].endRange
			}

			outFilesMap[index] = &outputFile{
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

		outFilesMap[0] = &outputFile{
			file: outfile,
		}

		defer outfile.Close()
	}

	for _, peer := range t.Peers {
		go t.startWorker(peer, workQueue, results)
	}

	for i, pieceHash := range t.PieceHashes {
		length := t.PieceLength

		if i == len(t.PieceHashes)-1 && t.Length%t.PieceLength != 0 {
			length = t.Length % t.PieceLength
		}

		workQueue <- &pieceWork{
			index:  i,
			length: length,
			hash:   pieceHash,
		}

	}

	var piecesDownloaded int

	for piecesDownloaded < len(t.PieceHashes) {
		piece := <-results

		piecesDownloaded++

		if !isMultifile {
			offset := piece.index * t.PieceLength
			_, err := outFilesMap[0].file.WriteAt(piece.data, int64(offset))

			if err != nil {
				return err
			}

		} else {

			err := piece.writeToFiles(outFilesMap, t.PieceLength)

			if err != nil {
				return err
			}
		}

		progressbar.Update(piecesDownloaded)
	}

	close(workQueue)

	return nil
}

func (p *pieceResult) writeToFiles(files map[int]*outputFile, pieceLength int) error {

	pieceOffsetStart := p.index * pieceLength
	pieceOffsetEnd := pieceOffsetStart + p.length

	bytesWritten := 0

	keysToDelete := make([]int, 0)

	var sortedKeys []int

	for key := range files {
		sortedKeys = append(sortedKeys, key)
	}

	sort.Slice(sortedKeys, func(i, j int) bool {
		return i < j
	})

	for _, key := range sortedKeys {
		file := files[key]
		if pieceOffsetEnd > file.startRange && pieceOffsetStart < file.endRange {
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

			// flag file for delete from map if all bytes are written
			if file.remainingBytes == 0 {
				keysToDelete = append(keysToDelete, key)
			}

			if bytesWritten == p.length {
				break
			}
		}
	}

	for _, key := range keysToDelete {
		delete(files, key)
	}

	return nil
}

func (t *Torrent) startWorker(peer peer.Peer, workQueue chan *pieceWork, resultsChan chan *pieceResult) {
	client, err := client.New(peer, t.PeerId, t.InfoHash)

	if err != nil {
		return
	}
	defer client.Conn.Close()

	client.SendUnchokeMsg()
	client.SendInterestedMsg()

	for work := range workQueue {

		if !client.BitField.HasPiece(work.index) {
			workQueue <- work
			continue
		}

		buffer, err := downloadPiece(client, work)

		if err != nil {
			// fmt.Printf("Failed to download piece %d from peer %s: %s\n", work.index, peer.Address, err.Error())
			workQueue <- work
			return
		}

		// check if hashes are same
		hash := sha1.Sum(buffer)

		if !bytes.Equal(hash[:], work.hash[:]) {
			// fmt.Printf("Piece %d from %s has incorrect hash\n", work.index, peer.Address)
			workQueue <- work
			continue
		}

		client.SendHaveMsg(work.index)
		resultsChan <- &pieceResult{
			index:  work.index,
			length: work.length,
			data:   buffer,
		}
	}
}

func downloadPiece(c *client.Client, work *pieceWork) ([]byte, error) {

	var numBlocks, numBlockRecieved, backlog, requested int

	if work.length%maxBlockSize == 0 {
		numBlocks = work.length / maxBlockSize
	} else {
		numBlocks = work.length/maxBlockSize + 1
	}

	blocksData := make([][]byte, numBlocks)

	c.Conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer c.Conn.SetDeadline(time.Time{}) // Disable the deadline

	for numBlockRecieved < numBlocks {

		if !c.Choked {
			for backlog < maxBacklog && requested < work.length {
				blockSize := maxBlockSize

				if work.length-requested < maxBlockSize {
					blockSize = work.length - requested
				}

				err := c.SendRequestMsg(work.index, requested, blockSize)

				if err != nil {
					return nil, err
				}

				requested += blockSize
				backlog++
			}
		}

		msg, err := message.Read(c.Conn)

		if err != nil {
			return nil, err
		}

		// Keep alive message recieved
		if msg == nil {
			continue
		}

		switch msg.ID {
		case message.UnchokeMessageID:
			c.Choked = false
		case message.ChokeMessageID:
			c.Choked = true
		case message.PieceMessageID:
			blockIndex := int(binary.BigEndian.Uint32(msg.Payload[4:8])) / maxBlockSize
			blocksData[blockIndex] = msg.Payload[8:]

			numBlockRecieved++
			backlog--
		case message.HaveMessageID:
			index := int(binary.BigEndian.Uint32(msg.Payload))
			c.BitField.SetPiece(index)
		}

	}

	return bytes.Join(blocksData, nil), nil
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
