package main

import (
	// Uncomment this line to pass the first stage
	// "encoding/json"
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bencode"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/metainfo"
)

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

		conn, err := net.Dial("tcp", os.Args[3])

		if err != nil {
			fmt.Println("Failed to connect to peer: " + err.Error())
			os.Exit(1)
		}

		handShakeMsg := make([]byte, 0)

		handShakeMsg = append(handShakeMsg, 19) // Length of the protocol string
		handShakeMsg = append(handShakeMsg, []byte("BitTorrent protocol")...)
		handShakeMsg = append(handShakeMsg, make([]byte, 8)...)                // 8 reserved bytes
		handShakeMsg = append(handShakeMsg, metaInfo.InfoHash...)              // Info hash
		handShakeMsg = append(handShakeMsg, []byte("00112233445566778899")...) // Peer ID

		conn.Write(handShakeMsg)

		responseBuffer := make([]byte, 68)
		_, err = conn.Read(responseBuffer)

		if err != nil {
			fmt.Println("Failed to read handshake response: " + err.Error())
			os.Exit(1)
		}

		fmt.Printf("Peer ID: %x\n", responseBuffer[48:68])

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}

}
