package p2p

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/client"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/message"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
)

type messageResult struct {
	id   byte
	data []byte
}

func (t *Torrent) startWorker(peer peer.Peer, workQueue chan *pieceWork, resultsChan chan *pieceResult) {
	client, err := client.New(peer, t.PeerId, t.InfoHash, len(t.PieceHashes))

	if err != nil {
		// fmt.Printf("Failed to create client for peer %s: %s\n", peer.Address, err.Error())
		return
	}
	defer client.Conn.Close()

	client.SendUnchokeMsg()
	client.SendInterestedMsg()

	closeChan := make(chan struct{})
	defer close(closeChan)

	//NOTE: buffered channel length should be decided
	messageChan := make(chan *messageResult, 30)

	go parseMessageFromConn(client, messageChan, closeChan)

	for work := range workQueue {

		if !client.BitField.HasPiece(work.index) {
			workQueue <- work
			continue
		}

		buffer, err := downloadPiece(client, work, messageChan)

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

		client.SendHaveMsg(work.index)

		resultsChan <- &pieceResult{
			index:  work.index,
			length: work.length,
			data:   buffer,
		}
	}
}

func downloadPiece(c *client.Client, work *pieceWork, messageChan chan *messageResult) ([]byte, error) {

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

		if msg.id == message.PieceMessageID {
			blockIndex := int(binary.BigEndian.Uint32(msg.data[4:8])) / maxBlockSize
			blocksData[blockIndex] = msg.data[8:]

			numBlockRecieved++
			backlog--
		}

	}

	return bytes.Join(blocksData, nil), nil
}

func parseMessageFromConn(client *client.Client, messageResultChan chan *messageResult, closeChan chan struct{}) {

	// Setting a deadline to get rid of unresponsive peers
	// 20 seconds is more than enough time to download a block
	client.Conn.SetReadDeadline(time.Now().Add(20 * time.Second))

	for {
		select {
		case <-closeChan:
			return
		default:
			msg, err := message.Read(client.Conn)

			if err != nil {
				messageResultChan <- nil
				return
			}

			if msg == nil {
				// Keep alive message recieved
				continue
			}

			result := messageResult{
				id: msg.ID,
			}

			switch msg.ID {
			case message.UnchokeMessageID:
				client.Choked = false
			case message.ChokeMessageID:
				client.Choked = true
			case message.HaveMessageID:
				index := int(binary.BigEndian.Uint32(msg.Payload))
				client.BitField.SetPiece(index)
			case message.BitfieldMessageID:
				client.BitField = msg.Payload
			case message.ExtensionMessageId:
				// TODO: Handle extension messages
			case message.PieceMessageID:
				result.data = msg.Payload
				// extend the read deadline
				client.Conn.SetReadDeadline(time.Now().Add(20 * time.Second))
			}

			messageResultChan <- &result
		}
	}
}
