package p2p

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/client"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/message"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
)

const (
	maxBlockSize = 16384
	maxBacklog   = 5
)

type Torrent struct {
	Peers       []peer.Peer
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	PeerId      []byte
}

type pieceWork struct {
	index  int
	length int
	hash   [20]byte
}

type pieceResult struct {
	index int
	data  []byte
}

func (t *Torrent) Download(outpath string) error {
	workQueue := make(chan *pieceWork, len(t.PieceHashes))
	results := make(chan *pieceResult)

	outfile, err := os.Create(outpath)

	if err != nil {
		return err
	}

	defer outfile.Close()

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

		offset := piece.index * t.PieceLength
		_, err := outfile.WriteAt(piece.data, int64(offset))

		if err != nil {
			return err
		}

		percent := float64(piecesDownloaded) / float64(len(t.PieceHashes)) * 100
		fmt.Printf("(%0.2f%%) Downloaded piece #%d from peers\n", percent, piece.index)
	}

	close(workQueue)

	return nil
}

func (t *Torrent) startWorker(peer peer.Peer, workQueue chan *pieceWork, resultsChan chan *pieceResult) {
	client, err := client.New(peer, t.PeerId, t.InfoHash)

	if err != nil {
		fmt.Printf("Failed to handshake with peer: %s \n", peer.Address)
		return
	}
	defer client.Conn.Close()
	fmt.Println("Handshake successful with peer: ", peer.Address)

	client.SendUnchokeMsg()
	client.SendInterestedMsg()

	for work := range workQueue {

		if !client.BitField.HasPiece(work.index) {
			workQueue <- work
			continue
		}

		buffer, err := downloadPiece(client, work)

		if err != nil {
			fmt.Printf("Failed to download piece %d from peer %s: %s\n", work.index, peer.Address, err.Error())
			workQueue <- work
			return
		}

		// check if hashes are same
		hash := sha1.Sum(buffer)

		if !bytes.Equal(hash[:], work.hash[:]) {
			fmt.Printf("Piece %d from %s has incorrect hash\n", work.index, peer.Address)
			fmt.Printf("Expected: %xand got %x\n", work.hash, hash)
			workQueue <- work
			continue
		}

		client.SendHaveMsg(work.index)
		resultsChan <- &pieceResult{
			index: work.index,
			data:  buffer,
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
