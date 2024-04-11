package tracker

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
)

type UPD_TRACKER_ACTION int

const (
	UPD_CONNECT_ACTION UPD_TRACKER_ACTION = iota
	UPD_ANNOUNCE_ACTION
)

func getPeersFromUDPTracker(baseUrl *url.URL, infoHash, peerId []byte, length int) ([]peer.Peer, error) {

	socket, err := net.Dial("udp", baseUrl.Host)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to tracker: %s", err.Error())
	}

	defer socket.Close()

	// connect request
	buffer := make([]byte, 16)

	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)
	transactionID := random.Uint32()
	var connectionId uint64

	binary.BigEndian.PutUint64(buffer[0:8], 0x41727101980)   // protocol_id
	binary.BigEndian.PutUint32(buffer[8:12], 0)              // action
	binary.BigEndian.PutUint32(buffer[12:16], transactionID) // transaction_id

	_, err = socket.Write(buffer)

	if err != nil {
		return nil, fmt.Errorf("failed to send connect request: %s", err.Error())
	}

	time.Sleep(2 * time.Second)

	socket.Write(buffer)

	response := make([]byte, 256)

	for {

		n, err := socket.Read(response)

		if err != nil {
			return nil, fmt.Errorf("failed to read from socket: %s", err.Error())
		}

		action := respType(response[:n])

		switch action {
		case UPD_CONNECT_ACTION:
			// handle connect response
			fmt.Println("Connect response", n)
			transactionIdRespone := binary.BigEndian.Uint32(response[4:8])
			connectionId = binary.BigEndian.Uint64(response[8:16])

			if transactionIdRespone != transactionID {
				return nil, fmt.Errorf("transaction id mismatch")
			}

			announceReq := buildAnnounceRequest(connectionId, transactionID, infoHash, peerId, length)

			_, err = socket.Write(announceReq)

			if err != nil {
				return nil, fmt.Errorf("failed to send announce request: %s", err.Error())
			}

		case UPD_ANNOUNCE_ACTION:
			// handle announce response
			fmt.Println("Announce response", n)

			peersBytes, err := parseAnnounceResponse(response[:n], transactionID)

			if err != nil {
				return nil, fmt.Errorf("failed to parse announce response: %s", err.Error())
			}

			return peer.Unmarshal(peersBytes)

		default:
			fmt.Println("Unknown action", action)
		}

	}

}

func buildAnnounceRequest(connectionId uint64, transactionId uint32, infoHash, peerId []byte, length int) []byte {
	buffer := make([]byte, 98)

	binary.BigEndian.PutUint64(buffer[0:8], connectionId)                 // connection_id
	binary.BigEndian.PutUint32(buffer[8:12], uint32(UPD_ANNOUNCE_ACTION)) // action
	binary.BigEndian.PutUint32(buffer[12:16], transactionId)              // transaction_id
	copy(buffer[16:36], infoHash)                                         // info_hash
	copy(buffer[36:56], peerId)                                           // peer_id
	binary.BigEndian.PutUint64(buffer[56:64], 0)                          // downloaded
	binary.BigEndian.PutUint64(buffer[64:72], uint64(length))             // left
	binary.BigEndian.PutUint64(buffer[72:80], 0)                          // uploaded
	binary.BigEndian.PutUint32(buffer[80:84], 0)                          // event
	binary.BigEndian.PutUint32(buffer[84:88], 0)                          // ip
	binary.BigEndian.PutUint32(buffer[88:92], 0)                          // key
	binary.BigEndian.PutUint32(buffer[92:96], uint32(0xFFFFFFFF))         // num_want
	binary.BigEndian.PutUint16(buffer[96:98], 6881)                       // port

	return buffer
}

func parseAnnounceResponse(buffer []byte, tranactionId uint32) ([]byte, error) {
	if len(buffer) < 20 {
		return nil, fmt.Errorf("invalid response")
	}

	tranactionIdResponse := binary.BigEndian.Uint32(buffer[4:8])

	if tranactionIdResponse != tranactionId {
		return nil, fmt.Errorf("transaction id mismatch")
	}

	peerBytes := buffer[20:]

	return peerBytes, nil
}

func respType(buf []byte) UPD_TRACKER_ACTION {
	return UPD_TRACKER_ACTION(binary.BigEndian.Uint32(buf[0:4]))
}
