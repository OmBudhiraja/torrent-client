package magnetlink

import (
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/OmBudhiraja/torrent-client/internal/p2p"
	"github.com/OmBudhiraja/torrent-client/internal/peer"
	"github.com/OmBudhiraja/torrent-client/internal/torrentfile"
	"github.com/OmBudhiraja/torrent-client/internal/tracker"
	"github.com/OmBudhiraja/torrent-client/pkg/progressbar"
	"github.com/zeebo/bencode"
)

const (
	supportedInfoTypes = "urn:btih:"
)

type MagnetLink struct {
	torrent  *p2p.Torrent
	dsm      *p2p.DownloadSessionManger
	infoHash [20]byte
	peerId   []byte
	peers    []peer.Peer

	metadataBytesChan      chan []byte
	isMetataDownloadedChan chan struct{}
	torrentInitailizedChan chan struct{}
}

func New(magnetUrl string, peerId []byte) (*MagnetLink, error) {
	parsedUrl, err := url.Parse(magnetUrl)

	if err != nil {
		return nil, err
	}

	trackerUrl := parsedUrl.Query().Get("tr")

	if trackerUrl == "" {
		return nil, fmt.Errorf("dht magnet links are not supported yet")
	}

	infoType := parsedUrl.Query().Get("xt")

	if !strings.HasPrefix(infoType, supportedInfoTypes) {
		return nil, fmt.Errorf("unsupported info type: %s", infoType)
	}

	hash, err := hex.DecodeString(strings.TrimPrefix(infoType, supportedInfoTypes))

	if err != nil {
		return nil, fmt.Errorf("failed to decode info hash: %s", err.Error())
	}

	infoHash := [20]byte{}
	copy(infoHash[:], hash)

	fmt.Printf("Waiting for peers...")
	peers, err := tracker.GetPeers(trackerUrl, infoHash, peerId, math.MaxInt)

	if err != nil {
		fmt.Println()
		return nil, err
	}

	fmt.Printf("\rFound %d peers           \n", len(peers))

	magnetLink := &MagnetLink{
		infoHash:               infoHash,
		peerId:                 peerId,
		peers:                  peers,
		metadataBytesChan:      make(chan []byte),
		isMetataDownloadedChan: make(chan struct{}),
		torrentInitailizedChan: make(chan struct{}),
	}

	return magnetLink, nil
}

func (magnetLink *MagnetLink) Download(outpath string) error {

	for _, peer := range magnetLink.peers {
		go handlePeer(peer, magnetLink)
	}

	// wait until one peer has completed metadata download
	mt := <-magnetLink.metadataBytesChan
	close(magnetLink.isMetataDownloadedChan)

	err := magnetLink.initializeTorrentFromMetadata(mt, outpath)

	if err != nil {
		return fmt.Errorf("failed to load torrent metadata: %s", err.Error())
	}

	progressbar := progressbar.New(len(magnetLink.torrent.PieceHashes))

	progressbar.Start()
	defer progressbar.Finish()

	dsm, err := magnetLink.torrent.Initiate()
	magnetLink.dsm = dsm
	defer dsm.CloseFiles()

	downloadedPices := 0

	close(magnetLink.torrentInitailizedChan)

	if err != nil {
		return fmt.Errorf("failed to initiate torrent download: %s", err.Error())
	}

	for downloadedPices < len(magnetLink.torrent.PieceHashes) {
		piece := <-dsm.Results
		downloadedPices++

		err := piece.WriteToFiles(dsm.PieceToFileMap[piece.Index], magnetLink.torrent.PieceLength)

		if err != nil {
			return fmt.Errorf("failed to write piece to file: %s", err.Error())
		}

		progressbar.Update(downloadedPices)
	}

	close(dsm.WorkQueue)

	return nil
}

func (magnetLink *MagnetLink) initializeTorrentFromMetadata(metadata []byte, outpath string) error {
	var info torrentfile.BencodeInfo

	err := bencode.DecodeBytes(metadata, &info)

	if err != nil {
		return fmt.Errorf("failed to decode metadata: %s", err.Error())
	}

	pieceHashes, err := info.PieceHashes()

	if err != nil {
		return fmt.Errorf("failed to get piece hashes: %s", err.Error())
	}

	_, files := info.IsMultiFile()

	// Create torrent file from metadata
	t := &p2p.Torrent{
		InfoHash:    magnetLink.infoHash,
		PieceHashes: pieceHashes,
		PieceLength: info.PieceLength,
		Length:      info.Length,
		Name:        info.Name,
		Files:       files,
		PeerId:      magnetLink.peerId,
		Peers:       magnetLink.peers,
		Outpath:     outpath,
	}

	magnetLink.torrent = t

	return nil
}
