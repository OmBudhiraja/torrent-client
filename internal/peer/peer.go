package peer

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/jackpal/bencode-go"
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

type bencodeTrackerResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

const (
	peerSize = 6
)

func Request(announce string, peerId string, infoHash []byte, length int) ([]Peer, error) {

	trackerUrl, err := buildTrackerUrl(announce, peerId, infoHash, length)

	if err != nil {
		return nil, err
	}

	resp, err := http.Get(trackerUrl)

	if err != nil {
		return nil, fmt.Errorf("failed to get peers from tracker: %s", err.Error())
	}

	defer resp.Body.Close()

	trackerResp := bencodeTrackerResponse{}

	err = bencode.Unmarshal(resp.Body, &trackerResp)

	if err != nil {
		return nil, fmt.Errorf("failed to decode peers response: %s", err.Error())
	}

	return unmarshal([]byte(trackerResp.Peers))
}

func buildTrackerUrl(announce string, peerId string, infoHash []byte, length int) (string, error) {

	baseUrl, err := url.Parse(announce)

	if err != nil {
		return "", fmt.Errorf("failed to parse tracker url: %s", err.Error())
	}

	params := url.Values{}

	params.Add("info_hash", string(infoHash))
	params.Add("peer_id", peerId)
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", fmt.Sprintf("%d", length))
	params.Add("compact", "1")

	baseUrl.RawQuery = params.Encode()

	return baseUrl.String(), nil
}

func unmarshal(data []byte) ([]Peer, error) {
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
