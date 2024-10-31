package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/magnetlink"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
)

func main() {

	peerId := []byte("00112233445566778899")

	var useMagnetLink bool
	flag.BoolVar(&useMagnetLink, "m", false, "Use magnet link instead of torrent file")

	flag.Parse()

	if useMagnetLink {
		tf, err := magnetlink.New(flag.Arg(0), peerId)

		if err != nil {
			fmt.Println("Failed to parse magnet link: " + err.Error())
			os.Exit(1)
		}

		err = tf.Download(os.Args[3])

		if err != nil {
			fmt.Println("Failed to download file: " + err.Error())
			os.Exit(1)
		}

		fmt.Println("Downloaded file")

		return
	}

	if len(os.Args) < 3 {
		fmt.Println("Usage: mybittorrent <torrent filepath> <output path>")
		os.Exit(1)
	}

	inFile := os.Args[1]
	outFile := os.Args[2]

	tf, err := torrentfile.New(inFile, peerId)

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

	fmt.Printf("Downloaded %s to %s in %s\n", inFile, outFile, time.Since(now).Round(time.Second).String())

}
