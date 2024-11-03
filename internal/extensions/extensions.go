package extensions

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/OmBudhiraja/torrent-client/internal/message"
	"github.com/zeebo/bencode"
)

const (
	ExtensionHandshakeId byte = 0
)

var (
	supportedExtensions = map[string]int{
		"ut_metadata": 1, // Metadata extension id for our peer
	}
)

type extensionHandshakeT struct {
	M            map[string]int `bencode:"m"`
	MetadataSize int            `bencode:"metadata_size"`
}

func SendHandshakeMessage(conn net.Conn) error {

	bencodedDictionary := extensionHandshakeT{
		M: supportedExtensions,
	}

	extensionsListBytes, err := bencode.EncodeBytes(bencodedDictionary)

	if err != nil {
		return fmt.Errorf("failed to encode extensions list: %s", err.Error())
	}

	payload := make([]byte, 0)
	payload = append(payload, 0) // message id is 0 for handshake message
	payload = append(payload, extensionsListBytes...)

	length := uint32(len(payload) + 1)
	handshakeMsg := make([]byte, 4+length)

	binary.BigEndian.PutUint32(handshakeMsg[:4], length)
	handshakeMsg[4] = message.ExtensionMessageId

	copy(handshakeMsg[5:], payload)

	_, err = conn.Write(handshakeMsg)

	if err != nil {
		return fmt.Errorf("failed to send extension handshake message: %s", err.Error())
	}

	return nil
}

func ParseHandshakeMessage(data []byte) (*extensionHandshakeT, error) {

	id := data[0]
	payload := data[1:]

	// If id is 0, then it is a handshake message
	if id != ExtensionHandshakeId {
		return nil, nil
	}

	var extensions extensionHandshakeT
	err := bencode.DecodeBytes(payload, &extensions)

	if err != nil {
		return nil, fmt.Errorf("failed to decode extension handshake message: %s", err.Error())
	}

	return &extensions, nil
}
