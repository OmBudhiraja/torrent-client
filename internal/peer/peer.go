package peer

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/zeebo/bencode"
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
	if data[0] == 'l' {
		return unmarshalNonCompact(data)
	} else {
		var out []byte
		err := bencode.DecodeString(string(data), &out)
		if err != nil {
			return nil, fmt.Errorf("failed to decode peers response: %s", err.Error())
		}
		return unmarshalCompact(out)
	}
}

func unmarshalCompact(data []byte) ([]Peer, error) {
	numPeers := len(data) / peerSize

	if len(data)%peerSize != 0 {
		fmt.Println(len(data)%peerSize, len(data), data)
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

type uncompactPeer struct {
	Ip   string `bencode:"ip"`
	Port int    `bencode:"port"`
}

func unmarshalNonCompact(data []byte) ([]Peer, error) {

	var decodedPeers []uncompactPeer

	err := bencode.DecodeBytes(data, &decodedPeers)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal peers: %s", err.Error())
	}

	peers := make([]Peer, len(decodedPeers))

	for i, p := range decodedPeers {
		peers[i] = Peer{Address: fmt.Sprintf("%s:%d", p.Ip, p.Port)}
	}

	return peers, nil
}
