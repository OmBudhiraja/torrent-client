package tracker

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"time"

	"github.com/OmBudhiraja/torrent-client/internal/peer"
)

type UPD_TRACKER_ACTION int

const (
	UPD_CONNECT_ACTION UPD_TRACKER_ACTION = iota
	UPD_ANNOUNCE_ACTION

	PROTOCOL_ID         uint64 = 0x41727101980
	MAX_RETRIES                = 8
	INITIAL_RETRY_DELAY        = 15 * time.Second
)

func getPeersFromUDPTracker(baseUrl *url.URL, infoHash, peerId []byte, length int) ([]peer.Peer, error) {
	socket, err := net.Dial("udp", baseUrl.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tracker: %s", err.Error())
	}
	defer socket.Close()

	var connectionId uint64
	var transactionID uint32

	// Set a deadline for the entire operation
	socket.SetDeadline(time.Now().Add(5 * time.Minute))

	// Initial connect request with retries
	connectionId, err = sendConnectRequestWithRetry(socket)
	if err != nil {
		return nil, fmt.Errorf("failed to establish connection with tracker: %s", err.Error())
	}

	// Send announce request with retries
	transactionID = createTransactionId()
	announceReq := buildAnnounceRequest(connectionId, transactionID, infoHash, peerId, length)

	peers, err := sendAnnounceRequestWithRetry(socket, announceReq, transactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get peers from tracker: %s", err.Error())
	}

	return peers, nil
}

func sendConnectRequestWithRetry(socket net.Conn) (uint64, error) {
	var connectionId uint64
	var transactionID uint32

	for retry := 0; retry <= MAX_RETRIES; retry++ {
		transactionID = createTransactionId()
		err := sendConnectRequest(socket, transactionID)
		if err != nil {
			return 0, err
		}

		// Wait for response with timeout
		response := make([]byte, 1024)
		socket.SetReadDeadline(time.Now().Add(INITIAL_RETRY_DELAY * time.Duration(1<<retry)))

		n, err := socket.Read(response)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Retry on timeout
			}
			return 0, err
		}

		if n < 16 {
			continue // Invalid response, retry
		}

		action := respType(response[:n])
		if action != UPD_CONNECT_ACTION {
			continue // Wrong response type, retry
		}

		transactionIdResponse := binary.BigEndian.Uint32(response[4:8])
		if transactionIdResponse != transactionID {
			continue // Transaction ID mismatch, retry
		}

		connectionId = binary.BigEndian.Uint64(response[8:16])
		return connectionId, nil
	}

	return 0, fmt.Errorf("failed to establish connection after %d retries", MAX_RETRIES)
}

func sendAnnounceRequestWithRetry(socket net.Conn, announceReq []byte, transactionID uint32) ([]peer.Peer, error) {
	for retry := 0; retry <= MAX_RETRIES; retry++ {
		_, err := socket.Write(announceReq)
		if err != nil {
			return nil, err
		}

		// Wait for response with timeout
		response := make([]byte, 1024)
		socket.SetReadDeadline(time.Now().Add(INITIAL_RETRY_DELAY * time.Duration(1<<retry)))

		n, err := socket.Read(response)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Retry on timeout
			}
			return nil, err
		}

		if n < 20 {
			continue // Invalid response, retry
		}

		action := respType(response[:n])
		if action != UPD_ANNOUNCE_ACTION {
			continue // Wrong response type, retry
		}

		peersBytes, err := parseAnnounceResponse(response[:n], transactionID)
		if err != nil {
			continue // Invalid response, retry
		}

		return peer.Unmarshal(peersBytes)
	}

	return nil, fmt.Errorf("failed to get peers after %d retries", MAX_RETRIES)
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

func sendConnectRequest(socket net.Conn, transactionID uint32) error {
	buffer := make([]byte, 16)

	binary.BigEndian.PutUint64(buffer[0:8], PROTOCOL_ID)     // protocol_id
	binary.BigEndian.PutUint32(buffer[8:12], 0)              // action
	binary.BigEndian.PutUint32(buffer[12:16], transactionID) // transaction_id

	_, err := socket.Write(buffer)

	if err != nil {
		return fmt.Errorf("failed to send connect request: %s", err.Error())
	}

	return nil
}

func respType(buf []byte) UPD_TRACKER_ACTION {
	return UPD_TRACKER_ACTION(binary.BigEndian.Uint32(buf[0:4]))
}

func createTransactionId() uint32 {
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)
	return random.Uint32()
}
