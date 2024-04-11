package main

import (
	"fmt"
	"os"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
)

func main() {

	if len(os.Args) < 3 {
		fmt.Println("Usage: mybittorrent <torrent filepath> <output path>")
		os.Exit(1)
	}

	inFile := os.Args[1]
	outFile := os.Args[2]

	tf, err := torrentfile.New(inFile)

	if err != nil {
		fmt.Println("Failed to parse torrent file: " + err.Error())
		os.Exit(1)
	}

	// fmt.Println(tf)
	// return

	now := time.Now()

	err = tf.Download(outFile)

	if err != nil {
		fmt.Println("Failed to download file: " + err.Error())
		os.Exit(1)
	}

	fmt.Printf("Downloaded %s to %s in %s\n", inFile, outFile, time.Since(now).Round(time.Second).String())

}
