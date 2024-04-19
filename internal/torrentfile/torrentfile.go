package torrentfile

import (
	"crypto/sha1"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"io"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/p2p"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/tracker"
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
}

type bencodeInfo struct {
	Name        string `bencode:"name"`
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Files       []file `bencode:"files"`
}

type file struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
}

type bencodeTorrent struct {
	Announce string             `bencode:"announce"`
	Info     bencode.RawMessage `bencode:"info"`
	info     bencodeInfo
}

func New(path string) (*TorrentFile, error) {
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

	infoDict := bencodeInfo{}

	err = bencode.DecodeBytes(bencodeTo.Info, &infoDict)

	if err != nil {
		return nil, err
	}

	bencodeTo.info = infoDict

	pieceHashes, err := bencodeTo.info.pieceHashes()

	if err != nil {
		return nil, err
	}

	isMultiFile := len(bencodeTo.info.Files) > 0

	var files []p2p.File

	if isMultiFile {

		for _, file := range bencodeTo.info.Files {
			bencodeTo.info.Length += file.Length
			fileParts := make([]string, len(file.Path))

			for i, p := range file.Path {
				fileParts[i] = cleanName(p)
			}

			files = append(files, p2p.File{
				Length: file.Length,
				Path:   filepath.Join(fileParts...),
			})

		}
	}

	return &TorrentFile{
		Announce:    bencodeTo.Announce,
		InfoHash:    infoHash,
		PieceHashes: pieceHashes,
		PieceLength: bencodeTo.info.PieceLength,
		Length:      bencodeTo.info.Length,
		Name:        bencodeTo.info.Name,
		Files:       files,
		IsMultiFile: isMultiFile,
	}, nil
}

func (t *TorrentFile) Download(outpath string) error {

	peerId := []byte("00112233445566778899")

	fmt.Printf("Waiting for peers...")
	peers, err := tracker.GetPeers(t.Announce, peerId, t.InfoHash[:], t.Length)
	fmt.Printf("\rFound %d peers           \n", len(peers))

	if err != nil {
		return err
	}

	if len(peers) == 0 {
		return fmt.Errorf("no peers found")
	}

	torrent := p2p.Torrent{
		Peers:       peers,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		PeerId:      peerId,
		Files:       t.Files,
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
	sb.WriteString(fmt.Sprintf("Is Multi File: %t\n", t.IsMultiFile))

	if t.IsMultiFile {
		sb.WriteString("Files: \n")
		for _, file := range t.Files {
			sb.WriteString(fmt.Sprintf("  - %s (%d bytes)\n", file.Path, file.Length))
		}

	}

	return sb.String()
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

// utils

func cleanName(s string) string {
	s = strings.ToValidUTF8(s, string(unicode.ReplacementChar))
	s = trimName(s, 255)
	s = strings.ToValidUTF8(s, "")
	return replaceSeparator(s)
}

func trimName(s string, max int) string {
	if len(s) <= max {
		return s
	}
	ext := path.Ext(s)
	if len(ext) > max {
		return s[:max]
	}
	return s[:max-len(ext)] + ext
}

func replaceSeparator(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' {
			return '_'
		}
		return r
	}, s)
}
