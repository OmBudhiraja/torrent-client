package main

import (
	// Uncomment this line to pass the first stage
	// "encoding/json"
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"math/rand"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bencode"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/metainfo"
)

type Peer struct {
	value string
	mu    sync.Mutex
}

func getPeers(metaInfo *metainfo.MetaInfo) ([]*Peer, error) {
	params := url.Values{}

	params.Add("info_hash", string(metaInfo.InfoHash))
	params.Add("peer_id", "00112233445566778899")
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", fmt.Sprintf("%d", metaInfo.Length))
	params.Add("compact", "1")

	resp, err := http.Get(fmt.Sprintf("%s?%s", metaInfo.TrackerUrl, params.Encode()))

	if err != nil {
		return nil, fmt.Errorf("failed to get peers from tracker: %s", err.Error())

	}

	defer resp.Body.Close()

	trackerResponseRaw, err := bencode.Decode(bufio.NewReader(resp.Body))

	if err != nil {
		return nil, fmt.Errorf("failed to decode peers response: %s", err.Error())
	}

	trackerResponse, ok := trackerResponseRaw.(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("failed to convert tracker response to dictionary")

	}

	peers, ok := trackerResponse["peers"].(string)

	if !ok {
		return nil, fmt.Errorf("failed to convert peers to string")
	}

	peersBytes := []byte(peers)

	result := make([]*Peer, len(peersBytes)/6)

	for i := 0; i < len(peersBytes); i += 6 {
		ip := fmt.Sprintf("%d.%d.%d.%d", peersBytes[i], peersBytes[i+1], peersBytes[i+2], peersBytes[i+3])
		port := binary.BigEndian.Uint16(peersBytes[i+4 : i+6])
		result[i/6] = &Peer{value: fmt.Sprintf("%s:%d", ip, port)}
	}

	return result, nil
}

func handshakePeer(peer string, infoHash []byte) (net.Conn, error) {
	conn, err := net.Dial("tcp", peer)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to peer: %s", err.Error())
	}

	handShakeMsg := make([]byte, 0)

	handShakeMsg = append(handShakeMsg, 19) // Length of the protocol string
	handShakeMsg = append(handShakeMsg, []byte("BitTorrent protocol")...)
	handShakeMsg = append(handShakeMsg, make([]byte, 8)...) // 8 reserved bytes

	handShakeMsg = append(handShakeMsg, infoHash...)                       // Info hash
	handShakeMsg = append(handShakeMsg, []byte("00112233445566778899")...) // Peer ID

	_, err = conn.Write(handShakeMsg)

	if err != nil {
		return nil, fmt.Errorf("failed to send handshake message: %s", err.Error())
	}

	return conn, nil
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

		metaInfo, err := metainfo.Parse(torrentFile)

		if err != nil {
			fmt.Println("Failed to parse torrent file: " + err.Error())
			os.Exit(1)
		}

		metaInfo.Print()

	case "peers":
		if len(os.Args) < 3 {
			fmt.Println("No torrent file provided")
			os.Exit(1)
		}

		torrentFile := os.Args[2]
		metaInfo, err := metainfo.Parse(torrentFile)

		if err != nil {
			fmt.Println("Failed to parse torrent file: " + err.Error())
			os.Exit(1)
		}

		peers, err := getPeers(metaInfo)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		for _, peer := range peers {
			fmt.Println(peer.value)
		}

	case "handshake":
		if len(os.Args) < 4 {
			fmt.Println("wrong number of arguments for handshake command")
			os.Exit(1)
		}

		torrentFile := os.Args[2]

		metaInfo, err := metainfo.Parse(torrentFile)

		if err != nil {
			fmt.Println("Failed to parse torrent file: " + err.Error())
			os.Exit(1)
		}

		conn, err := handshakePeer(os.Args[3], metaInfo.InfoHash)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		responseBuffer := make([]byte, 68)
		_, err = conn.Read(responseBuffer)

		if err != nil {
			fmt.Println("Failed to read handshake response: ", err)
			os.Exit(1)
		}

		fmt.Printf("Peer ID: %x\n", responseBuffer[48:68])

	case "download_piece":
		if len(os.Args) < 6 {
			fmt.Println("wrong number of arguments for download_piece command")
			os.Exit(1)
		}

		outputPath := os.Args[3]
		torrentFile := os.Args[4]
		pieceIndexStr := os.Args[5]

		metaInfo, err := metainfo.Parse(torrentFile)

		if err != nil {
			fmt.Println("Failed to parse torrent file: " + err.Error())
			os.Exit(1)
		}

		pieceIndex, err := strconv.Atoi(pieceIndexStr)

		if err != nil {
			fmt.Println("Invalid piece index: " + err.Error())
			os.Exit(1)
		}

		peers, err := getPeers(metaInfo)

		randomPeer := peers[rand.Intn(len(peers))]

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		res := downloadPiece(metaInfo, pieceIndex, randomPeer.value)

		outFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

		if err != nil {
			fmt.Println("Failed to open output file: " + err.Error())
			os.Exit(1)
		}

		defer outFile.Close()

		_, err = outFile.Write(res)

		if err != nil {
			fmt.Println("Failed to write to output file: " + err.Error())
			os.Exit(1)
		}

		fmt.Printf("Piece %d downloaded to %s\n", pieceIndex, outputPath)

	case "download":
		if len(os.Args) < 5 {
			fmt.Println("wrong number of arguments for download command")
			os.Exit(1)
		}

		outputPath := os.Args[3]
		torrentFile := os.Args[4]

		metaInfo, err := metainfo.Parse(torrentFile)

		if err != nil {
			fmt.Println("Failed to parse torrent file: " + err.Error())
			os.Exit(1)
		}

		peers, err := getPeers(metaInfo)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		outFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY, 0644)

		if err != nil {
			fmt.Println("Failed to open output file: " + err.Error())
			os.Exit(1)
		}

		defer outFile.Close()

		wg := sync.WaitGroup{}

		for i := 0; i < len(metaInfo.PieceHashes); i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				selectedPeer := peers[i%len(peers)]
				selectedPeer.mu.Lock()
				defer selectedPeer.mu.Unlock()

				res := downloadPiece(metaInfo, i, selectedPeer.value)
				writePiece(outFile, res, i, metaInfo.PieceLength)
			}(i)
		}

		wg.Wait()

		fmt.Printf("Downloaded %s to %s\n", torrentFile, outputPath)

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}

}

func writePiece(outfile *os.File, data []byte, index int, pieceLength int) {

	offset := index * pieceLength

	_, err := outfile.WriteAt(data, int64(offset))

	if err != nil {
		fmt.Printf("Failed to write piece with idx: %i to output file: "+err.Error(), index)
		os.Exit(1)
	}
}

const (
	ChokeMessageID byte = iota
	UnchokeMessageID
	InterestedMessageID
	NotInterestedMessageID
	HaveMessageID
	BitfieldMessageID
	RequestMessageID
	PieceMessageID
	CancelMessageID
)

const KB16 = 16384

func downloadPiece(metaInfo *metainfo.MetaInfo, pieceIndex int, peer string) []byte {

	pieceLength := metaInfo.PieceLength

	if pieceIndex == len(metaInfo.PieceHashes)-1 {
		pieceLength = metaInfo.Length % pieceLength
	}

	conn, err := handshakePeer(peer, metaInfo.InfoHash)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer conn.Close()

	// fmt.Println("Downloading piece ", pieceIndex, " from ", peer, "with length ", pieceLength)

	responseBuffer := make([]byte, 68)
	// _, err = io.ReadFull(conn, responseBuffer)
	_, err = conn.Read(responseBuffer)

	if err != nil {
		fmt.Println("Failed to read handshake response: "+err.Error(), pieceIndex)
		os.Exit(1)
	}

	messageLengthBytes := make([]byte, 4)
	messageIdByte := make([]byte, 1)

	var numBlocks int

	if pieceLength%KB16 == 0 {
		numBlocks = metaInfo.PieceLength / KB16
	} else {
		numBlocks = int(math.Floor(float64(pieceLength)/KB16)) + 1
	}

	blocksData := make([][]byte, numBlocks)
	numsBlockReceived := 0

outerLoop:
	for {
		n, err := io.ReadFull(conn, messageLengthBytes)

		if err != nil {
			fmt.Println("Failed to read message: " + err.Error())
			os.Exit(1)
		}

		if n != len(messageLengthBytes) {
			fmt.Println("Invalid message length")
			os.Exit(1)
		}

		payloadSize := int(binary.BigEndian.Uint32(messageLengthBytes))

		if payloadSize == 0 {
			fmt.Println("Keep alive message")
			continue
		}

		_, err = io.ReadFull(conn, messageIdByte)

		if err != nil {
			fmt.Println("Failed to read message ID: " + err.Error())
			os.Exit(1)
		}

		payloadSize--
		payload := make([]byte, payloadSize)

		switch messageIdByte[0] {
		case BitfieldMessageID:
			_, err := io.ReadFull(conn, payload)

			if err != nil {
				fmt.Println("Failed to read bitfield: " + err.Error())
				os.Exit(1)
			}

			// send interested message
			conn.Write([]byte{0, 0, 0, 1, InterestedMessageID})

		case UnchokeMessageID:
			for i := 0; i < numBlocks; i++ {
				// send request message
				payload := make([]byte, 12)

				blockSize := KB16

				if pieceLength%KB16 != 0 && i == numBlocks-1 {
					blockSize = pieceLength % KB16
				}

				binary.BigEndian.PutUint32(payload[:4], uint32(pieceIndex))  // piece index
				binary.BigEndian.PutUint32(payload[4:8], uint32(i*KB16))     // block begin offset
				binary.BigEndian.PutUint32(payload[8:12], uint32(blockSize)) // block length

				conn.Write(append([]byte{0, 0, 0, 13, RequestMessageID}, payload...))
			}

		case PieceMessageID:
			// read piece message
			_, err = io.ReadFull(conn, payload)

			if err != nil {
				fmt.Println("Failed to read piece message: " + err.Error())
				os.Exit(1)
			}

			blockIndex := binary.BigEndian.Uint32(payload[4:8]) / KB16
			blockData := payload[8:]

			blocksData[blockIndex] = blockData
			numsBlockReceived++

			if numsBlockReceived == numBlocks {
				break outerLoop
			}

		case ChokeMessageID:
			fmt.Println("Choked")
			time.Sleep(3 * time.Second)
		default:
			fmt.Println("Unknown message ID: ", messageIdByte[0])
		}
	}

	combinedPieceData := make([]byte, 0)

	// combine all blocks
	for _, blockData := range blocksData {
		combinedPieceData = append(combinedPieceData, blockData...)
	}

	// integrity check
	sha1Hash := sha1.New()
	sha1Hash.Write(combinedPieceData)

	if bytes.Equal(metaInfo.InfoHash, sha1Hash.Sum(nil)) {
		fmt.Println("Integrity check failed")
		os.Exit(1)
	}

	return combinedPieceData
}
