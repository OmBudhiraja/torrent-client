package p2p

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"

	"github.com/OmBudhiraja/torrent-client/internal/client"
	"github.com/OmBudhiraja/torrent-client/internal/message"
	"github.com/OmBudhiraja/torrent-client/internal/peer"
)

func (t *Torrent) StartWorker(peer peer.Peer, workQueue chan *PieceWork, resultsChan chan *PieceResult) {
	peerClient, err := client.New(peer, t.InfoHash, t.PeerId, len(t.PieceHashes))

	if err != nil {
		// fmt.Printf("Failed to create client for peer %s: %s\n", peer.Address, err.Error())
		return
	}
	defer peerClient.Conn.Close()

	closeChan := make(chan struct{})
	defer close(closeChan)

	//NOTE: buffered channel length should be decided
	messageChan := make(chan *client.MessageResult, 30)

	go peerClient.ParsePeerMessage(messageChan, closeChan)

	t.ResumeWorker(peerClient, workQueue, resultsChan, messageChan, closeChan)
}

func (t *Torrent) ResumeWorker(c *client.Client, workQueue chan *PieceWork, resultsChan chan *PieceResult, messageChan chan *client.MessageResult, closeChan chan struct{}) {
	c.SendUnchokeMsg()
	c.SendInterestedMsg()

	for work := range workQueue {
		if !c.BitField.HasPiece(work.Index) {
			workQueue <- work
			continue
		}

		buffer, err := DownloadPiece(c, work, messageChan)

		if err != nil {
			// fmt.Printf("Failed to download piece %d from peer %s: %s\n", work.Index, c.Peer.Address, err.Error())
			workQueue <- work
			closeChan <- struct{}{}
			return
		}

		// check if hashes are same
		hash := sha1.Sum(buffer)

		if !bytes.Equal(hash[:], work.Hash[:]) {
			// fmt.Printf("Piece %d from %s has incorrect hash\n", work.Index, c.Peer.Address)
			workQueue <- work
			continue
		}

		c.SendHaveMsg(work.Index)

		resultsChan <- &PieceResult{
			Index:  work.Index,
			Length: work.Length,
			Data:   buffer,
		}
	}
}

func DownloadPiece(c *client.Client, work *PieceWork, messageChan chan *client.MessageResult) ([]byte, error) {

	var numBlocks, numBlockRecieved, backlog, requested int

	if work.Length%maxBlockSize == 0 {
		numBlocks = work.Length / maxBlockSize
	} else {
		numBlocks = work.Length/maxBlockSize + 1
	}

	blocksData := make([][]byte, numBlocks)

	for numBlockRecieved < numBlocks {

		if !c.Choked {
			for backlog < maxBacklog && requested < work.Length {
				blockSize := maxBlockSize

				if work.Length-requested < maxBlockSize {
					blockSize = work.Length - requested
				}

				err := c.SendRequestMsg(work.Index, requested, blockSize)

				if err != nil {
					return nil, err
				}

				requested += blockSize
				backlog++
			}
		}

		msg := <-messageChan

		if msg.Err != nil {
			return nil, msg.Err
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
