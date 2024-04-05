package torrentfile

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/p2p"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
	"github.com/jackpal/bencode-go"
)

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

type bencodeInfo struct {
	Name        string `bencode:"name"`
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

func New(path string) (*TorrentFile, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("Failed to open torrent file: " + err.Error())
	}

	defer file.Close()

	bencodeTo := bencodeTorrent{}

	err = bencode.Unmarshal(file, &bencodeTo)

	if err != nil {
		return nil, err
	}

	infoHash, err := bencodeTo.Info.hash()

	if err != nil {
		return nil, err
	}

	pieceHashes, err := bencodeTo.Info.pieceHashes()

	if err != nil {
		return nil, err
	}

	return &TorrentFile{
		Announce:    bencodeTo.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bencodeTo.Info.PieceLength,
		Length:      bencodeTo.Info.Length,
		Name:        bencodeTo.Info.Name,
	}, nil
}

func (t *TorrentFile) Download(outpath string) error {

	peerId := "00112233445566778899"

	peers, err := peer.Request(t.Announce, peerId, t.InfoHash[:], t.Length)

	if err != nil {
		return err
	}

	fmt.Println("Got peers: ", peers)

	torrent := p2p.Torrent{
		Peers:       peers,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		PeerId:      []byte(peerId),
	}

	err = torrent.Download(outpath)

	if err != nil {
		return err
	}

	return nil
}

func (t *TorrentFile) String() string {

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Name: %s\n", t.Name))
	sb.WriteString(fmt.Sprintf("Tracker URL: %s\n", t.Announce))
	sb.WriteString(fmt.Sprintf("Length: %d\n", t.Length))
	sb.WriteString(fmt.Sprintf("Info Hash: %x\n", t.InfoHash))
	sb.WriteString(fmt.Sprintf("Piece Length: %d\n", t.PieceLength))
	sb.WriteString(fmt.Sprintf("No. of Piece: %d\n", len(t.PieceHashes)))

	return sb.String()
}

func (info *bencodeInfo) hash() ([20]byte, error) {
	var buffer bytes.Buffer
	err := bencode.Marshal(&buffer, *info)
	if err != nil {
		return [20]byte{}, err
	}

	return sha1.Sum(buffer.Bytes()), nil
}

func (info *bencodeInfo) pieceHashes() ([][20]byte, error) {
	hashLength := 20
	pieceHashesBytes := []byte(info.Pieces)

	if len(pieceHashesBytes)%hashLength != 0 {
		return nil, fmt.Errorf("invalid pieces length %d", len(pieceHashesBytes))
	}

	hashes := [][20]byte{}
	for i := 0; i < len(pieceHashesBytes); i += 20 {
		var hash [20]byte
		copy(hash[:], pieceHashesBytes[i:i+20])
		hashes = append(hashes, hash)
	}

	return hashes, nil
}
