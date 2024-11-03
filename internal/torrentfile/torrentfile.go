package torrentfile

import (
	"crypto/sha1"
	"fmt"
	"os"

	"io"

	"github.com/OmBudhiraja/torrent-client/internal/p2p"
	"github.com/OmBudhiraja/torrent-client/internal/tracker"
	"github.com/zeebo/bencode"
)

type TorrentFile struct {
	Announce    string
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
	Files       []p2p.File
	IsMultiFile bool
	PeerId      []byte
}

type file struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
}

type bencodeTorrent struct {
	Announce string             `bencode:"announce"`
	Info     bencode.RawMessage `bencode:"info"`
	info     BencodeInfo
}

func New(path string, peerId []byte) (*TorrentFile, error) {
	file, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("Failed to open torrent file: " + err.Error())
	}

	defer file.Close()

	bencodeTo := bencodeTorrent{}

	filedata, err := io.ReadAll(file)

	if err != nil {
		return nil, err
	}

	err = bencode.DecodeBytes(filedata, &bencodeTo)

	if err != nil {
		return nil, err
	}

	infoHash := sha1.Sum(bencodeTo.Info)

	infoDict := BencodeInfo{}

	err = bencode.DecodeBytes(bencodeTo.Info, &infoDict)

	if err != nil {
		return nil, err
	}

	bencodeTo.info = infoDict

	pieceHashes, err := bencodeTo.info.PieceHashes()

	if err != nil {
		return nil, err
	}

	isMultiFile, files := bencodeTo.info.IsMultiFile()

	return &TorrentFile{
		Announce:    bencodeTo.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bencodeTo.info.PieceLength,
		Length:      bencodeTo.info.Length,
		Name:        bencodeTo.info.Name,
		Files:       files,
		IsMultiFile: isMultiFile,
		PeerId:      peerId,
	}, nil
}

func (t *TorrentFile) Download(outpath string) error {

	fmt.Printf("Waiting for peers...")
	peers, err := tracker.GetPeers(t.Announce, t.InfoHash, t.PeerId, t.Length)

	if err != nil {
		fmt.Println()
		return err
	}

	fmt.Printf("\rFound %d peers           \n", len(peers))

	if len(peers) == 0 {
		return fmt.Errorf("no peers found")
	}

	torrent := p2p.Torrent{
		Name:        t.Name,
		Peers:       peers,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		PeerId:      t.PeerId,
		Files:       t.Files,
		Outpath:     outpath,
	}

	err = torrent.Download()

	if err != nil {
		return err
	}

	return nil
}
