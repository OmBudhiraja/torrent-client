package torrentfile

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/OmBudhiraja/torrent-client/internal/p2p"
)

type BencodeInfo struct {
	Name        string `bencode:"name"`
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Files       []file `bencode:"files"`
}

func (info *BencodeInfo) PieceHashes() ([][20]byte, error) {
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

func (info *BencodeInfo) IsMultiFile() (isMultiFile bool, files []p2p.File) {
	isMultiFile = len(info.Files) > 0

	if isMultiFile {
		for _, file := range info.Files {
			info.Length += file.Length
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

	return isMultiFile, files
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
