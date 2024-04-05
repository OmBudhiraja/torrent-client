package main

import (
	"fmt"
	"os"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
)

func main() {

	if len(os.Args) < 3 {
		fmt.Println("Usage: mybittorrent <torrent file> <output file>")
		os.Exit(1)
	}

	inFile := os.Args[1]
	outFile := os.Args[2]

	tf, err := torrentfile.New(inFile)

	if err != nil {
		fmt.Println("Failed to parse torrent file: " + err.Error())
		os.Exit(1)
	}

	now := time.Now()

	err = tf.Download(outFile)

	if err != nil {
		fmt.Println("Failed to download file: " + err.Error())
		os.Exit(1)
	}

	fmt.Printf("Downloaded %s to %s in %.2fs\n", inFile, outFile, time.Since(now).Minutes())

}
