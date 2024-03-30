package main

import (
	// Uncomment this line to pass the first stage
	// "encoding/json"
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/bencode"
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
		file, err := os.Open(torrentFile)

		if err != nil {
			fmt.Println("Failed to open torrent file: " + err.Error())
			os.Exit(1)
		}

		reader := bufio.NewReader(file)

		res, err := bencode.Decode(reader)
		if err != nil {
			fmt.Println(err)
			return
		}

		decoded, ok := res.(map[string]interface{})

		if !ok {
			fmt.Println("Failed to convert decoded value to dictionary")
			os.Exit(1)
		}

		info, ok := decoded["info"].(map[string]interface{})

		if !ok {
			fmt.Println("No info dictionary found in torrent file")
			os.Exit(1)
		}

		// convert info dictionary to sha256 hash
		sha1Hash := sha1.New()
		sha1Hash.Write([]byte(bencode.Encode(info)))

		fmt.Println("Tracker URL:", decoded["announce"])
		fmt.Println("Length:", info["length"])
		fmt.Printf("Info Hash: %s\n", hex.EncodeToString(sha1Hash.Sum(nil)))

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}

}
