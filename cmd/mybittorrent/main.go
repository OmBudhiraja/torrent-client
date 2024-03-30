package main

import (
	// Uncomment this line to pass the first stage
	// "encoding/json"
	"bufio"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bencode"
)

type TorrentInfo struct {
	TrackerUrl  string
	InfoHash    []byte
	Length      int
	PieceLength int
	PieceHashes []string
}

func (t *TorrentInfo) Print() {
	fmt.Println("Tracker URL:", t.TrackerUrl)
	fmt.Println("Length:", t.Length)
	fmt.Printf("Info Hash: %x\n", t.InfoHash)
	fmt.Println("Piece Length:", t.PieceLength)
	fmt.Println("Piece Hashes:")

	for _, pieceHash := range t.PieceHashes {
		fmt.Println(pieceHash)
	}
}

func NewTorrentInfo(reader *bufio.Reader) (*TorrentInfo, error) {
	res, err := bencode.Decode(reader)
	if err != nil {
		return nil, err
	}

	decoded, ok := res.(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("failed to convert decoded value to dictionary")
	}

	info, ok := decoded["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no info dictionary found in torrent file")
	}

	sha1Hash := sha1.New()
	sha1Hash.Write([]byte(bencode.Encode(info)))

	pieceHashes := []string{}
	pieceHashesBytes := []byte(info["pieces"].(string))

	for i := 0; i < len(pieceHashesBytes); i += 20 {
		pieceHashes = append(pieceHashes, fmt.Sprintf("%x", pieceHashesBytes[i:i+20]))
	}

	return &TorrentInfo{
		TrackerUrl:  decoded["announce"].(string),
		InfoHash:    sha1Hash.Sum(nil),
		Length:      info["length"].(int),
		PieceLength: info["piece length"].(int),
		PieceHashes: pieceHashes,
	}, nil
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("No command provided")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "decode":
		if len(os.Args) < 3 {
			fmt.Println("No bencoded value provided")
			os.Exit(1)
		}
		bencodedValue := os.Args[2]
		reader := bufio.NewReader(strings.NewReader(bencodedValue))

		decoded, err := bencode.Decode(reader)

		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	case "info":
		if len(os.Args) < 3 {
			fmt.Println("No torrent file provided")
			os.Exit(1)
		}
		torrentFile := os.Args[2]
		file, err := os.Open(torrentFile)

		if err != nil {
			fmt.Println("Failed to open torrent file: " + err.Error())
			os.Exit(1)
		}

		reader := bufio.NewReader(file)

		torrentInfo, err := NewTorrentInfo(reader)

		if err != nil {
			fmt.Println("Failed to parse torrent file: " + err.Error())
			os.Exit(1)
		}

		torrentInfo.Print()

	case "peers":
		if len(os.Args) < 3 {
			fmt.Println("No torrent file provided")
			os.Exit(1)
		}

		torrentFile := os.Args[2]
		file, err := os.Open(torrentFile)

		if err != nil {
			fmt.Println("Failed to open torrent file: " + err.Error())
			os.Exit(1)
		}

		reader := bufio.NewReader(file)

		torrentInfo, err := NewTorrentInfo(reader)

		if err != nil {
			fmt.Println("Failed to parse torrent file: " + err.Error())
			os.Exit(1)
		}

		params := url.Values{}

		params.Add("info_hash", string(torrentInfo.InfoHash))
		params.Add("peer_id", "00112233445566778899")
		params.Add("port", "6881")
		params.Add("uploaded", "0")
		params.Add("downloaded", "0")
		params.Add("left", fmt.Sprintf("%d", torrentInfo.Length))
		params.Add("compact", "1")

		resp, err := http.Get(fmt.Sprintf("%s?%s", torrentInfo.TrackerUrl, params.Encode()))

		if err != nil {
			fmt.Println("Failed to get peers from tracker: " + err.Error())
			os.Exit(1)
		}

		defer resp.Body.Close()

		trackerResponseRaw, err := bencode.Decode(bufio.NewReader(resp.Body))

		if err != nil {
			fmt.Println("Failed to decode peers response: " + err.Error())
			os.Exit(1)
		}

		trackerResponse, ok := trackerResponseRaw.(map[string]interface{})

		if !ok {
			fmt.Println("Failed to convert tracker response to dictionary")
			os.Exit(1)
		}

		peers, ok := trackerResponse["peers"].(string)

		if !ok {
			fmt.Println("Failed to convert peers to string")
			os.Exit(1)
		}

		peersBytes := []byte(peers)

		for i := 0; i < len(peersBytes); i += 6 {
			ip := fmt.Sprintf("%d.%d.%d.%d", peersBytes[i], peersBytes[i+1], peersBytes[i+2], peersBytes[i+3])
			port := binary.BigEndian.Uint16(peersBytes[i+4 : i+6])
			fmt.Printf("%s:%d\n", ip, port)
		}

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}

}
