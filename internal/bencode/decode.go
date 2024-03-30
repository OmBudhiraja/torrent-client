package bencode

import (
	"bufio"
	"fmt"
	"strconv"
	"unicode"
)

func Decode(reader *bufio.Reader) (interface{}, error) {

	firstChar, err := reader.ReadByte()

	if err != nil {
		return "", fmt.Errorf("failed to read first byte: %v", err)
	}

	switch {
	case unicode.IsDigit(rune(firstChar)):
		return decodeString(reader)
	case firstChar == 'i':
		return decodeInteger(reader)
	case firstChar == 'l':
		return decodeList(reader)
	case firstChar == 'd':
		return decodeDictionary(reader)
	default:
		return "", fmt.Errorf("unsupported bencode type with byte: %s", string(firstChar))
	}
}

func decodeString(reader *bufio.Reader) (string, error) {

	// unread the first byte to read the length of the string
	err := reader.UnreadByte()

	if err != nil {
		return "", fmt.Errorf("failed to unread first byte for string: %v", err)
	}

	n, err := reader.ReadBytes(':')

	if err != nil {
		return "", fmt.Errorf("failed to read length of string: %v", err)
	}

	length, err := strconv.Atoi(string(n[:len(n)-1]))

	if err != nil {
		return "", fmt.Errorf("failed to convert length to integer: %v", err)
	}

	bencodedString := make([]byte, length)

	_, err = reader.Read(bencodedString)

	if err != nil {
		return "", fmt.Errorf("failed to read string: %v", err)
	}

	if len(bencodedString) != length {
		return "", fmt.Errorf("string length mismatch: expected %d, got %d", length, len(bencodedString))
	}

	return string(bencodedString), nil
}

func decodeInteger(reader *bufio.Reader) (int, error) {

	intStr, err := reader.ReadBytes('e')

	if err != nil {
		return 0, fmt.Errorf("failed to read integer: %v", err)
	}

	integer, err := strconv.Atoi(string(intStr[:len(intStr)-1]))

	if err != nil {
		return 0, fmt.Errorf("failed to convert integer: %v", err)
	}

	return integer, nil
}

func decodeList(reader *bufio.Reader) ([]interface{}, error) {

	list := make([]interface{}, 0)

	for {
		nextChar, err := reader.Peek(1)

		if err != nil {
			return nil, fmt.Errorf("failed to peek next byte: %v", err)
		}

		if nextChar[0] == 'e' {
			_, err := reader.ReadByte()

			if err != nil {
				return nil, fmt.Errorf("failed to read list end: %v", err)
			}

			return list, nil
		}

		decoded, err := Decode(reader)

		if err != nil {
			return nil, fmt.Errorf("failed to decode list element: %v", err)
		}

		list = append(list, decoded)

	}
}

func decodeDictionary(reader *bufio.Reader) (map[string]interface{}, error) {

	dict := make(map[string]interface{})

	list, err := decodeList(reader)

	if err != nil {
		return nil, fmt.Errorf("failed to decode dictionary: %v", err)
	}

	if len(list)%2 != 0 {
		return nil, fmt.Errorf("dictionary key value mismatch: %d", len(list))
	}

	for i := 0; i < len(list); i += 2 {
		key, ok := list[i].(string)

		if !ok {
			return nil, fmt.Errorf("failed to convert dictionary key to string: %v", list[i])
		}

		dict[key] = list[i+1]
	}

	return dict, nil
}
