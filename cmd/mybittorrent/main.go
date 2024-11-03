package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/OmBudhiraja/torrent-client/internal/magnetlink"
	"github.com/OmBudhiraja/torrent-client/internal/torrentfile"
)

type Downloader interface {
	Download(outFile string) error
}

func main() {

	downloader, err := getDownloader()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	now := time.Now()

	var outPath string

	if len(flag.Args()) < 2 {
		outPath = "."
	} else {
		outPath = flag.Args()[1]
	}

	err = downloader.Download(outPath)

	if err != nil {
		fmt.Println("Failed to download file: " + err.Error())
		os.Exit(1)
	}

	fmt.Printf("Successfully Downloaded to %s in %s\n", outPath, time.Since(now).Round(time.Second).String())

}

func getDownloader() (Downloader, error) {
	peerId := []byte("00112233445566778899")

	var useMagnetLink bool
	flag.BoolVar(&useMagnetLink, "m", false, "Use magnet link instead of torrent file")

	flag.Parse()

	opts := flag.Args()

	if len(opts) == 0 {
		fmt.Println("Usage: mybittorrent <torrent filepath> <output path>")
		os.Exit(1)
	}

	if useMagnetLink {
		mg, err := magnetlink.New(opts[0], peerId)

		if err != nil {
			return nil, fmt.Errorf("failed to parse magnet link: %s", err.Error())
		}

		return mg, nil
	}

	if len(os.Args) < 3 {
		fmt.Println("Usage: mybittorrent <torrent filepath> <output path>")
		os.Exit(1)
	}

	tf, err := torrentfile.New(opts[0], peerId)

	if err != nil {
		return nil, fmt.Errorf("failed to parse torrent file: %s", err.Error())
	}

	return tf, nil
}
