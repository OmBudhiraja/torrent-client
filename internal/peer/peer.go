package peer

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/zeebo/bencode"
)

const (
	PROTOCOL_NAME_HEADER = "BitTorrent protocol"
)

type Peer struct {
	Address string
}

type HandshakeResponse struct {
	Conn                      net.Conn
	SupportsExtensionProtocol bool
}

func (p Peer) CompleteHandshake(infoHash []byte, peerId []byte) (*HandshakeResponse, error) {
	conn, err := net.DialTimeout("tcp", p.Address, 5*time.Second)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer: %s", err.Error())
	}

	// asert that info hash and peer id are of the correct length
	if len(infoHash) != 20 {
		return nil, fmt.Errorf("invalid info hash length")
	}

	if len(peerId) != 20 {
		return nil, fmt.Errorf("invalid peer id length")
	}

	handshakeMsgSent := make([]byte, 0)

	handshakeMsgSent = append(handshakeMsgSent, 19) // Length of the protocol string
	handshakeMsgSent = append(handshakeMsgSent, []byte(PROTOCOL_NAME_HEADER)...)

	reservedByes := make([]byte, 8)

	// set the 20th bit to 1 to indicate that we support the extension protocol
	reservedByes[5] = 0x10

	handshakeMsgSent = append(handshakeMsgSent, reservedByes...) // 8 reserved bytes

	handshakeMsgSent = append(handshakeMsgSent, infoHash...) // Info hash
	handshakeMsgSent = append(handshakeMsgSent, peerId...)   // Peer ID

	_, err = conn.Write(handshakeMsgSent)

	if err != nil {
		return nil, fmt.Errorf("failed to send handshake message: %s", err.Error())
	}

	handshakeMsgRecieved := make([]byte, 68)

	n, err := io.ReadFull(conn, handshakeMsgRecieved)

	if err != nil {
		return nil, fmt.Errorf("failed to read handshake message: %s", err.Error())
	}

	if n < 68 {
		return nil, fmt.Errorf("invalid handshake message")
	}

	// verify handshake message
	if handshakeMsgRecieved[0] != 19 || string(handshakeMsgRecieved[1:20]) != PROTOCOL_NAME_HEADER {
		return nil, fmt.Errorf("invalid handshake message")
	}

	// check if the peer supports the extension protocol
	supportsExtensionProtocol := handshakeMsgRecieved[25]&0x10 == 0x10

	return &HandshakeResponse{
		Conn:                      conn,
		SupportsExtensionProtocol: supportsExtensionProtocol,
	}, nil
}

func Unmarshal(data []byte) ([]Peer, error) {

	if len(data) == 0 {
		return []Peer{}, nil
	}

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
	peerSize := 6
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
