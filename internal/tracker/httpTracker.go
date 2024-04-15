package tracker

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/zeebo/bencode"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
)

type bencodeTrackerResponse struct {
	Interval int                `bencode:"interval"`
	Peers    bencode.RawMessage `bencode:"peers"`
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

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %s", err.Error())
	}

	trackerResp := bencodeTrackerResponse{}

	err = bencode.DecodeBytes(respBody, &trackerResp)

	if err != nil {
		return nil, fmt.Errorf("failed to decode peers response: %s", err.Error())
	}

	return peer.Unmarshal(trackerResp.Peers)
}
