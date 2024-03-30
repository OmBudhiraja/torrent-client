package main

import (
	// Uncomment this line to pass the first stage
	// "encoding/json"
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

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

		decoded, err := decodeBencode(reader)

		if err != nil {
			return nil, fmt.Errorf("failed to decode list element: %v", err)
		}

		list = append(list, decoded)

	}
}

func decodeBencode(reader *bufio.Reader) (interface{}, error) {

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
	default:
		return "", fmt.Errorf("unsupported bencode type with byte: %s", string(firstChar))
	}
}

func main() {

	command := os.Args[1]

	if command == "decode" {

		bencodedValue := os.Args[2]
		reader := bufio.NewReader(strings.NewReader(bencodedValue))

		decoded, err := decodeBencode(reader)

		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
