package client

import (
	"encoding/binary"
	"net"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bitfield"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/extensions"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/message"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
)

type Client struct {
	Conn                      net.Conn
	Choked                    bool
	BitField                  bitfield.Bitfield
	Peer                      peer.Peer
	InfoHash                  [20]byte
	PeerId                    []byte
	SupportedExtension        map[string]int
	MetadataSize              int
	SupportsExtensionProtocol bool
}

func New(peer peer.Peer, infoHash [20]byte, peerId []byte, totalPieces int) (*Client, error) {
	handshakeRes, err := peer.CompleteHandshake(infoHash[:], peerId)

	if err != nil {
		return nil, err
	}

	if handshakeRes.SupportsExtensionProtocol {
		extensions.SendHandshakeMessage(handshakeRes.Conn)
	}

	client := &Client{
		Conn:                      handshakeRes.Conn,
		Choked:                    true,
		Peer:                      peer,
		PeerId:                    peerId,
		InfoHash:                  infoHash,
		BitField:                  bitfield.New(totalPieces),
		SupportsExtensionProtocol: handshakeRes.SupportsExtensionProtocol,
	}

	return client, nil

}

type MessageResult struct {
	Err  error
	Id   byte
	Data []byte
}

func (c *Client) ParsePeerMessage(messageResultChan chan *MessageResult, closeChan chan struct{}) {

	for {
		select {
		case <-closeChan:
			return
		default:
			msg, err := message.Read(c.Conn)

			if err != nil {
				messageResultChan <- &MessageResult{
					Err: err,
				}
				return
			}

			if msg == nil {
				// Keep alive message recieved
				continue
			}

			result := MessageResult{
				Id:   msg.ID,
				Data: msg.Payload,
			}

			switch msg.ID {
			case message.UnchokeMessageID:
				c.Choked = false
			case message.ChokeMessageID:
				c.Choked = true
			case message.HaveMessageID:
				index := int(binary.BigEndian.Uint32(msg.Payload))
				c.BitField.SetPiece(index)
			case message.BitfieldMessageID:
				c.BitField = msg.Payload
			case message.ExtensionMessageId:
				res, err := extensions.ParseHandshakeMessage(msg.Payload)

				if err != nil {
					messageResultChan <- &MessageResult{
						Err: err,
					}
					return
				}

				if res != nil {
					c.MetadataSize = res.MetadataSize
					c.SupportedExtension = res.M
				}

			case message.PieceMessageID:
				//
			}

			messageResultChan <- &result
		}
	}
}

func (c *Client) SendInterestedMsg() error {
	msg := message.Message{
		ID: message.InterestedMessageID,
	}
	_, err := c.Conn.Write(msg.Encode())

	return err
}

func (c *Client) SendUnchokeMsg() error {
	msg := message.Message{
		ID: message.UnchokeMessageID,
	}
	_, err := c.Conn.Write(msg.Encode())

	return err
}

func (c *Client) SendRequestMsg(index, begin, length int) error {
	msg := message.Message{
		ID:      message.RequestMessageID,
		Payload: message.FormatRequestPayload(index, begin, length),
	}
	_, err := c.Conn.Write(msg.Encode())

	return err
}

func (c *Client) SendHaveMsg(index int) error {
	msg := message.Message{
		ID:      message.HaveMessageID,
		Payload: message.FormatHavePayload(index),
	}
	_, err := c.Conn.Write(msg.Encode())

	return err
}
