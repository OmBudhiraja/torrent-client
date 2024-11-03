package tracker

import (
	"fmt"
	"net/url"

	"github.com/OmBudhiraja/torrent-client/internal/peer"
)

func GetPeers(announce string, infohash [20]byte, peerId []byte, length int) ([]peer.Peer, error) {

	baseUrl, err := url.Parse(announce)

	if err != nil {
		return nil, fmt.Errorf("failed to parse tracker url: %s", err.Error())
	}

	if baseUrl.Scheme == "udp" {
		return getPeersFromUDPTracker(baseUrl, infohash[:], peerId, length)
	} else {
		return getPeersFromHTTPTracker(baseUrl, infohash[:], peerId, length)
	}

}
