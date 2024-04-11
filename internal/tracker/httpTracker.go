package tracker

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
	"github.com/jackpal/bencode-go"
)

type bencodeTrackerResponse struct {
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}

func getPeersFromHTTPTracker(baseUrl *url.URL, infoHash, peerId []byte, length int) ([]peer.Peer, error) {
	params := url.Values{}

	params.Add("info_hash", string(infoHash))
	params.Add("peer_id", string(peerId))
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", fmt.Sprintf("%d", length))
	params.Add("compact", "1")

	baseUrl.RawQuery = params.Encode()

	resp, err := http.Get(baseUrl.String())

	if err != nil {
		return nil, fmt.Errorf("failed to get peers from tracker: %s", err.Error())
	}

	defer resp.Body.Close()

	trackerResp := bencodeTrackerResponse{}

	err = bencode.Unmarshal(resp.Body, &trackerResp)

	if err != nil {
		return nil, fmt.Errorf("failed to decode peers response: %s", err.Error())
	}

	return peer.Unmarshal([]byte(trackerResp.Peers))
}
