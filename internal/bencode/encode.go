package bencode

import "fmt"

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

	for key, item := range value {
		result += encodeString(key)
		result += Encode(item)
	}

	return result + "e"
}
