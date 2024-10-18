package message

import (
	"encoding/binary"
	"io"
	"net"
)

const (
	ChokeMessageID         byte = 0
	UnchokeMessageID       byte = 1
	InterestedMessageID    byte = 2
	NotInterestedMessageID byte = 3
	HaveMessageID          byte = 4
	BitfieldMessageID      byte = 5
	RequestMessageID       byte = 6
	PieceMessageID         byte = 7
	CancelMessageID        byte = 8
	ExtensionMessageId     byte = 20
)

type Message struct {
	ID      byte
	Payload []byte
}

func Read(conn net.Conn) (*Message, error) {
	messageLengthBytes := make([]byte, 4)

	_, err := io.ReadFull(conn, messageLengthBytes)

	if err != nil {
		return nil, err
	}

	length := int(binary.BigEndian.Uint32(messageLengthBytes))

	// Keep alive message recieved
	if length == 0 {
		return nil, nil
	}

	payload := make([]byte, length)

	_, err = io.ReadFull(conn, payload)

	if err != nil {
		return nil, err
	}

	return &Message{
		ID:      payload[0],
		Payload: payload[1:],
	}, nil
}

func (m *Message) Encode() []byte {

	// For Keep alive messages
	if m == nil {
		return make([]byte, 4)
	}

	length := uint32(len(m.Payload) + 1)
	message := make([]byte, 4+length)

	binary.BigEndian.PutUint32(message[:4], length)
	message[4] = m.ID

	copy(message[5:], m.Payload)

	return message
}

func FormatRequestPayload(index, begin, length int) []byte {
	payload := make([]byte, 12)

	binary.BigEndian.PutUint32(payload[:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	return payload
}

func FormatHavePayload(index int) []byte {
	payload := make([]byte, 4)

	binary.BigEndian.PutUint32(payload[:4], uint32(index))

	return payload
}
