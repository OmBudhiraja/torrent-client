package peer

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type Peer struct {
	Address string
}

func (p Peer) CompleteHandshake(infoHash []byte, peerId []byte) (net.Conn, error) {
	conn, err := net.Dial("tcp", p.Address)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer: %s", err.Error())
	}

	handShakeMsg := make([]byte, 0)

	handShakeMsg = append(handShakeMsg, 19) // Length of the protocol string
	handShakeMsg = append(handShakeMsg, []byte("BitTorrent protocol")...)
	handShakeMsg = append(handShakeMsg, make([]byte, 8)...) // 8 reserved bytes

	handShakeMsg = append(handShakeMsg, infoHash...) // Info hash
	handShakeMsg = append(handShakeMsg, peerId...)   // Peer ID

	_, err = conn.Write(handShakeMsg)

	if err != nil {
		return nil, fmt.Errorf("failed to send handshake message: %s", err.Error())
	}

	handshakeMsg := make([]byte, 68)

	_, err = io.ReadFull(conn, handshakeMsg)

	if err != nil {
		return nil, fmt.Errorf("failed to read handshake message: %s", err.Error())
	}

	return conn, nil
}

const (
	peerSize = 6
)

func Unmarshal(data []byte) ([]Peer, error) {
	numPeers := len(data) / peerSize

	if len(data)%peerSize != 0 {
		return nil, fmt.Errorf("invalid peer data")
	}

	peers := make([]Peer, numPeers)

	for i := 0; i < len(data); i += peerSize {
		ip := fmt.Sprintf("%d.%d.%d.%d", data[i], data[i+1], data[i+2], data[i+3])
		port := binary.BigEndian.Uint16(data[i+4 : i+6])
		peers[i/peerSize] = Peer{Address: fmt.Sprintf("%s:%d", ip, port)}
	}

	return peers, nil
}
