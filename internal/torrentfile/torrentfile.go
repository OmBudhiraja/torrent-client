package torrentfile

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"strings"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/p2p"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/tracker"
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
	Files       []File `bencode:"files"`
}

type File struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
	file     *os.File
}

func New(path string) (*TorrentFile, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("Failed to open torrent file: " + err.Error())
	}

	defer file.Close()

	bencodeTo := bencodeTorrent{
		file: file,
	}

	err = bencode.Unmarshal(file, &bencodeTo)

	if err != nil {
		return nil, err
	}

	infoHash, err := bencodeTo.hash()

	if err != nil {
		return nil, err
	}

	pieceHashes, err := bencodeTo.Info.pieceHashes()

	if err != nil {
		return nil, err
	}

	isMultiFile := len(bencodeTo.Info.Files) > 0

	if isMultiFile {
		for _, file := range bencodeTo.Info.Files {
			bencodeTo.Info.Length += file.Length
		}
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

	peerId := []byte("00112233445566778899")

	peers, err := tracker.GetPeers(t.Announce, peerId, t.InfoHash[:], t.Length)

	if err != nil {
		return err
	}

	if len(peers) == 0 {
		return fmt.Errorf("no peers found")
	}

	fmt.Println("Got peers: ", peers)

	torrent := p2p.Torrent{
		Peers:       peers,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		PeerId:      peerId,
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

func (bt *bencodeTorrent) hash() ([20]byte, error) {

	bt.file.Seek(0, 0)

	decodedTorrent, err := bencode.Decode(bt.file)

	if err != nil {
		return [20]byte{}, err
	}

	torrentDict, ok := decodedTorrent.(map[string]interface{})

	if !ok {
		return [20]byte{}, fmt.Errorf("invalid bencode format")
	}

	infoDict, ok := torrentDict["info"]

	if !ok {
		return [20]byte{}, fmt.Errorf("info key not found")
	}

	var buffer bytes.Buffer
	err = bencode.Marshal(&buffer, infoDict)

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
