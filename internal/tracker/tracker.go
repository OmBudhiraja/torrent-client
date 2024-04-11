package tracker

import (
	"fmt"
	"net/url"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
)

func GetPeers(announce string, peerId, infoHash []byte, length int) ([]peer.Peer, error) {

	baseUrl, err := url.Parse(announce)

	if err != nil {
		return nil, fmt.Errorf("failed to parse tracker url: %s", err.Error())
	}

	if baseUrl.Scheme == "udp" {
		return getPeersFromUDPTracker(baseUrl, infoHash, peerId, length)
	} else {
		return getPeersFromHTTPTracker(baseUrl, infoHash, peerId, length)
	}

}
