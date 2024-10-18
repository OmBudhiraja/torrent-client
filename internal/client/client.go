package client

import (
	"net"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bitfield"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/message"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
)

type Client struct {
	Conn     net.Conn
	Choked   bool
	BitField bitfield.Bitfield
	Peer     peer.Peer
	InfoHash [20]byte
	PeerId   []byte
}

func New(peer peer.Peer, peerId []byte, infoHash [20]byte, totalPieces int) (*Client, error) {
	handshakeRes, err := peer.CompleteHandshake(infoHash[:], peerId)

	if err != nil {
		return nil, err
	}

	client := &Client{
		Conn:     handshakeRes.Conn,
		Choked:   true,
		Peer:     peer,
		PeerId:   peerId,
		InfoHash: infoHash,
		BitField: bitfield.New(totalPieces),
	}

	return client, nil

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
