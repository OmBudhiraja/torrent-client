package magnetlink

import (
	"github.com/OmBudhiraja/torrent-client/internal/client"
	"github.com/OmBudhiraja/torrent-client/internal/extensions"
	"github.com/OmBudhiraja/torrent-client/internal/extensions/metadata"
	"github.com/OmBudhiraja/torrent-client/internal/message"
	"github.com/OmBudhiraja/torrent-client/internal/peer"
)

func handlePeer(peerClient peer.Peer, magnetLink *MagnetLink) {
	c, err := client.New(peerClient, magnetLink.infoHash, magnetLink.peerId, 0)

	if err != nil {
		return
	}

	messageResultChan := make(chan *client.MessageResult, 30)

	peerCloseChan := make(chan struct{})
	defer close(peerCloseChan)

	go c.ParsePeerMessage(messageResultChan, peerCloseChan)

	if c.SupportsExtensionProtocol {
		extensions.SendHandshakeMessage(c.Conn)
	}

	var fullMetadata []byte
	var downloadedMetadataSize int

outerLoop:
	for {

		select {
		case <-magnetLink.isMetataDownloadedChan:
			break outerLoop
		case msg := <-messageResultChan:

			if msg.Err != nil {
				// error reading message, close connection
				return
			}

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
						// fmt.Println("failed to format metadata request message payload", err)
						return
					}

					_, err = c.Conn.Write(metadataRequestMsg)

					if err != nil {
						// fmt.Println("failed to send metadata request message", err)
						return
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
					// fmt.Println("metadata size does not match???????")
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
						case magnetLink.metadataBytesChan <- fullMetadata:
						case <-magnetLink.isMetataDownloadedChan:
						}
						break outerLoop
					}

				}
			}
		}

	}

	for range magnetLink.torrentInitailizedChan {
		break
	}

	magnetLink.torrent.ResumeWorker(c, magnetLink.dsm.WorkQueue, magnetLink.dsm.Results, messageResultChan, peerCloseChan)
}
