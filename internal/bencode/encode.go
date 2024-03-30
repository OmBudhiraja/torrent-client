package bencode

import (
	"fmt"
	"sort"
)

func Encode(value interface{}) string {
	switch v := value.(type) {
	case string:
		return encodeString(v)
	case int:
		return encodeInteger(v)
	case []interface{}:
		return encodeList(v)
	case map[string]interface{}:
		return encodeDictionary(v)
	default:
		fmt.Println("Unsupported type????")
		return ""
	}
}

func encodeString(value string) string {
	return fmt.Sprintf("%d:%s", len(value), value)
}

func encodeInteger(value int) string {
	return fmt.Sprintf("i%de", value)
}

func encodeList(value []interface{}) string {
	result := "l"

	for _, item := range value {
		result += Encode(item)
	}

	return result + "e"
}

func encodeDictionary(value map[string]interface{}) string {
	result := "d"

	sortedKeys := make([]string, 0, len(value))

	for key := range value {
		sortedKeys = append(sortedKeys, key)
	}

	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		result += encodeString(key)
		result += Encode(value[key])
	}

	return result + "e"
}
