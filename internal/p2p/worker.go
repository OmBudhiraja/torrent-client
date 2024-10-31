package p2p

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/client"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/message"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
)

func (t *Torrent) startWorker(peer peer.Peer, workQueue chan *pieceWork, resultsChan chan *pieceResult) {
	peerClient, err := client.New(peer, t.InfoHash, t.PeerId, len(t.PieceHashes))

	if err != nil {
		// fmt.Printf("Failed to create client for peer %s: %s\n", peer.Address, err.Error())
		return
	}
	defer peerClient.Conn.Close()

	peerClient.SendUnchokeMsg()
	peerClient.SendInterestedMsg()

	closeChan := make(chan struct{})
	defer close(closeChan)

	//NOTE: buffered channel length should be decided
	messageChan := make(chan *client.MessageResult, 30)

	go peerClient.ParsePeerMessage(messageChan, closeChan)

	for work := range workQueue {

		if !peerClient.BitField.HasPiece(work.index) {
			workQueue <- work
			continue
		}

		buffer, err := downloadPiece(peerClient, work, messageChan)

		if err != nil {
			// fmt.Printf("Failed to download piece %d from peer %s: %s\n", work.index, peer.Address, err.Error())
			workQueue <- work
			closeChan <- struct{}{}
			return
		}

		// check if hashes are same
		hash := sha1.Sum(buffer)

		if !bytes.Equal(hash[:], work.hash[:]) {
			// fmt.Printf("Piece %d from %s has incorrect hash\n", work.index, peer.Address)
			workQueue <- work
			continue
		}

		peerClient.SendHaveMsg(work.index)

		resultsChan <- &pieceResult{
			index:  work.index,
			length: work.length,
			data:   buffer,
		}
	}
}

func downloadPiece(c *client.Client, work *pieceWork, messageChan chan *client.MessageResult) ([]byte, error) {

	var numBlocks, numBlockRecieved, backlog, requested int

	if work.length%maxBlockSize == 0 {
		numBlocks = work.length / maxBlockSize
	} else {
		numBlocks = work.length/maxBlockSize + 1
	}

	blocksData := make([][]byte, numBlocks)

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

		msg := <-messageChan

		if msg == nil {
			return nil, fmt.Errorf("failed to read message")
		}

		if msg.Id == message.PieceMessageID {
			blockIndex := int(binary.BigEndian.Uint32(msg.Data[4:8])) / maxBlockSize
			blocksData[blockIndex] = msg.Data[8:]

			numBlockRecieved++
			backlog--
		}

	}

	return bytes.Join(blocksData, nil), nil
}
