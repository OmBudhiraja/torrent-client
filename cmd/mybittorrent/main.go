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
	case firstChar == 'd':
		return decodeDictionary(reader)
	default:
		return "", fmt.Errorf("unsupported bencode type with byte: %s", string(firstChar))
	}
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("No command provided")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "decode":
		if len(os.Args) < 3 {
			fmt.Println("No bencoded value provided")
			os.Exit(1)
		}
		bencodedValue := os.Args[2]
		reader := bufio.NewReader(strings.NewReader(bencodedValue))

		decoded, err := decodeBencode(reader)

		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	case "info":
		if len(os.Args) < 3 {
			fmt.Println("No torrent file provided")
			os.Exit(1)
		}
		torrentFile := os.Args[2]
		file, err := os.Open(torrentFile)

		if err != nil {
			fmt.Println("Failed to open torrent file: " + err.Error())
			os.Exit(1)
		}

		reader := bufio.NewReader(file)

		res, err := decodeBencode(reader)
		if err != nil {
			fmt.Println(err)
			return
		}

		decoded, ok := res.(map[string]interface{})

		if !ok {
			fmt.Println("Failed to convert decoded value to dictionary")
			os.Exit(1)
		}

		info, ok := decoded["info"].(map[string]interface{})

		if !ok {
			fmt.Println("No info dictionary found in torrent file")
			os.Exit(1)
		}

		fmt.Println("Tracker URL:", decoded["announce"])
		fmt.Println("Length:", info["length"])

	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}

}
