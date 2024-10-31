package metadata

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/message"
	"github.com/zeebo/bencode"
)

const (
	ExtensionMessageRequestId byte = 0
	ExtensionMessageDataId    byte = 1
	ExtensionMessageRejectId  byte = 2

	MetadataExtensionId   byte = 1
	MetadataExtensionName      = "ut_metadata"

	PieceSize = 16384
)

type metadataMsgDict struct {
	MsgType   int `bencode:"msg_type"`
	Piece     int `bencode:"piece"`
	TotalSize int `bencode:"total_size,omitempty"`
}

type MetadataExtensionRes struct {
	MsgType   int
	Piece     int
	TotalSize int
	Data      []byte
}

func FormatRequestMsg(peerMetadataExtensionId, piece int) ([]byte, error) {

	payload := make([]byte, 0)

	payload = append(payload, byte(peerMetadataExtensionId))

	// dictData := map[string]int{
	// 	"msg_type": int(ExtensionMessageRequestId),
	// 	"piece":    piece,
	// }

	dictData := metadataMsgDict{
		MsgType: int(ExtensionMessageRequestId),
		Piece:   piece,
	}

	bencodedBytes, err := bencode.EncodeBytes(dictData)

	if err != nil {
		return nil, fmt.Errorf("failed to encode request message: %s", err.Error())
	}

	payload = append(payload, bencodedBytes...)

	length := len(payload) + 1

	msg := make([]byte, 4+length)

	binary.BigEndian.PutUint32(msg[:4], uint32(length))

	msg[4] = message.ExtensionMessageId
	copy(msg[5:], payload)

	return msg, nil

}

func HandleMetadataMsg(data []byte) (*MetadataExtensionRes, error) {
	decoder := bencode.NewDecoder(bytes.NewReader(data))

	var dataDict metadataMsgDict

	err := decoder.Decode(&dataDict)

	if err != nil {
		return nil, err
	}

	res := &MetadataExtensionRes{
		MsgType:   dataDict.MsgType,
		Piece:     dataDict.Piece,
		TotalSize: dataDict.TotalSize,
	}

	if dataDict.MsgType == int(ExtensionMessageRejectId) {
		return res, nil
	}

	dictEnd := decoder.BytesParsed()

	res.Data = data[dictEnd:]

	return res, nil

}
