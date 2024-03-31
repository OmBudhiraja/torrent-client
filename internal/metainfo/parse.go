package metainfo

import (
	"bufio"
	"crypto/sha1"
	"fmt"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bencode"
)

type MetaInfo struct {
	TrackerUrl  string
	InfoHash    []byte
	Length      int
	PieceLength int
	PieceHashes []string
}

func (t *MetaInfo) Print() {
	fmt.Println("Tracker URL:", t.TrackerUrl)
	fmt.Println("Length:", t.Length)
	fmt.Printf("Info Hash: %x\n", t.InfoHash)
	fmt.Println("Piece Length:", t.PieceLength)
	fmt.Println("Piece Hashes:")

	for _, pieceHash := range t.PieceHashes {
		fmt.Println(pieceHash)
	}
}

func Parse(fileName string) (*MetaInfo, error) {
	file, err := os.Open(fileName)

	if err != nil {
		return nil, fmt.Errorf("Failed to open torrent file: " + err.Error())
	}

	reader := bufio.NewReader(file)

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

	return &MetaInfo{
		TrackerUrl:  decoded["announce"].(string),
		InfoHash:    sha1Hash.Sum(nil),
		Length:      info["length"].(int),
		PieceLength: info["piece length"].(int),
		PieceHashes: pieceHashes,
	}, nil
}
