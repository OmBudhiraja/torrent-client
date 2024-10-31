package magnetlink

import (
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/client"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/extensions"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/extensions/metadata"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/message"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentfile"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/tracker"
	"github.com/zeebo/bencode"
)

const (
	supportedInfoTypes = "urn:btih:"
)

func New(magnetLink string, peerId []byte) (*torrentfile.TorrentFile, error) {
	parsedUrl, err := url.Parse(magnetLink)

	if err != nil {
		return nil, err
	}

	trackerUrl := parsedUrl.Query().Get("tr")

	infoType := parsedUrl.Query().Get("xt")

	if !strings.HasPrefix(infoType, supportedInfoTypes) {
		return nil, fmt.Errorf("unsupported info type: %s", infoType)
	}

	hash, err := hex.DecodeString(strings.TrimPrefix(infoType, supportedInfoTypes))

	if err != nil {
		return nil, fmt.Errorf("failed to decode info hash: %s", err.Error())
	}

	infoHash := [20]byte{}
	copy(infoHash[:], hash)

	peers, err := tracker.GetPeers(trackerUrl, infoHash, peerId, math.MaxInt)

	if err != nil {
		return nil, fmt.Errorf("failed to get peers: %s", err.Error())
	}

	metadataBytesChan := make(chan []byte)
	closeChan := make(chan struct{})

	for _, peer := range peers {
		go handlePeer(peer, infoHash, peerId, metadataBytesChan, closeChan)
	}

	select {
	case mt, ok := <-metadataBytesChan:
		if ok {
			// Signal all other goroutines to stop
			close(closeChan)

			var info torrentfile.BencodeInfo

			err := bencode.DecodeBytes(mt, &info)

			if err != nil {
				return nil, fmt.Errorf("failed to decode metadata: %s", err.Error())
			}

			pieceHashes, err := info.PieceHashes()

			if err != nil {
				return nil, fmt.Errorf("failed to get piece hashes: %s", err.Error())
			}

			name := info.Name

			if name == "" {
				name = "unknown"
			}

			// Create torrent file from metadata
			t := &torrentfile.TorrentFile{
				Announce:    trackerUrl,
				InfoHash:    infoHash,
				PieceHashes: pieceHashes,      // TODO: Parse from metadata
				PieceLength: info.PieceLength, // TODO: Parse from metadata
				Length:      info.Length,      // TODO: Parse from metadata
				Name:        name,             // TODO: Parse from metadata
				Files:       nil,              // TODO: Parse from metadata
				IsMultiFile: false,
				PeerId:      peerId,
			}

			return t, nil
		}
	}

	return nil, fmt.Errorf("failed to get metadata")
}

func handlePeer(peerClient peer.Peer, infoHash [20]byte, peerId []byte, metadataBytesChan chan []byte, closeChan chan struct{}) {
	c, err := client.New(peerClient, infoHash, peerId, 0)

	if err != nil {
		// TODO: handle error
	}

	messageResultChan := make(chan *client.MessageResult)

	peerCloseChan := make(chan struct{})
	defer close(peerCloseChan)

	go c.ParsePeerMessage(messageResultChan, peerCloseChan)

	if c.SupportsExtensionProtocol {
		extensions.SendHandshakeMessage(c.Conn)
	}

	var fullMetadata []byte
	var downloadedMetadataSize int

	for {

		select {
		case <-closeChan:
			return
		case msg := <-messageResultChan:

			if msg.Err != nil {
				// TODO: handle error
				return
			}

			fmt.Println("client ext", c.SupportedExtension, c.SupportsExtensionProtocol)

			extensionId := msg.Data[0]
			peerMetadataExtensionId := c.SupportedExtension[metadata.MetadataExtensionName]

			// extension handshake completed
			if msg.Id == message.ExtensionMessageId && extensionId == extensions.ExtensionHandshakeId {

				// send metadata request message if the peer supports metadata extension
				if peerMetadataExtensionId == 0 {
					// peer does not support metadata extension
					continue
				}

				var numPieces int

				if c.MetadataSize == 0 {
					numPieces = 1
				} else {
					numPieces = (c.MetadataSize + metadata.PieceSize - 1) / metadata.PieceSize
				}

				for i := 0; i < numPieces; i++ {
					metadataRequestMsg, err := metadata.FormatRequestMsg(peerMetadataExtensionId, i)

					if err != nil {
						fmt.Println("failed to format metadata request message payload", err)
						return // TODO: handle error
					}

					_, err = c.Conn.Write(metadataRequestMsg)

					if err != nil {
						fmt.Println("failed to send metadata request message", err)
						return // TODO: handle error
					}

				}

			}

			if msg.Id == message.ExtensionMessageId && extensionId == metadata.MetadataExtensionId {
				metadataRes, err := metadata.HandleMetadataMsg(msg.Data[1:])

				if err != nil {
					continue // peer sent invalid metadata message
				}

				if metadataRes.MsgType == int(metadata.ExtensionMessageRejectId) {
					// peer rejected the request
					continue
				}

				if metadataRes.MsgType == int(metadata.ExtensionMessageRequestId) {
					// currently not handling request message
					continue
				}

				if c.MetadataSize == 0 {
					c.MetadataSize = metadataRes.TotalSize
				} else if c.MetadataSize != 0 && c.MetadataSize != metadataRes.TotalSize {
					// metadata size does not match
					fmt.Println("metadata size does not match???????")
					continue
				}

				if len(fullMetadata) == 0 {
					fullMetadata = make([]byte, c.MetadataSize)
				}

				if metadataRes.MsgType == int(metadata.ExtensionMessageDataId) {

					start := metadataRes.Piece * metadata.PieceSize
					end := start + len(metadataRes.Data)

					copy(fullMetadata[start:end], metadataRes.Data)

					downloadedMetadataSize += len(metadataRes.Data)

					if downloadedMetadataSize == c.MetadataSize {
						// metadata download completed
						select {
						case metadataBytesChan <- fullMetadata:
						case <-closeChan:
						}
						return
					}

				}
			}
		}

	}

}

// func (m *Magnetlink)
