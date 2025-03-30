package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func Parse(r *bufio.Reader) (*Value, error) {
	firstByte, err := r.ReadByte()
	if err != nil {
		return &Value{}, fmt.Errorf("Unable to parse first byte::: %w", err)
	}

	switch firstByte {
	case '+': // SimpleString
		return parseSimpleString(r)
	case '$': // BulkString
		return parseBulkString(r)
	default:
		return nil, fmt.Errorf("Unknown RESP type::: %c", firstByte)
	}
}

func parseBulkString(r *bufio.Reader) (*Value, error) {
	strLen, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("Unable to read bulk string length::: %w", err)
	}
	strLen = strings.TrimSpace(strLen)
	length, err := strconv.Atoi(strLen)
	if err != nil {
		return nil, fmt.Errorf("Unable to convert bulk string length to integer::: %w", err)
	}

	// For RESP2 Compatibility
	// Null Bulk String
	if length == -1 {
		return &Value{
			Type:   BulkString,
			Length: length,
			String: "",
		}, nil
	}

	data := make([]byte, length)
	_, err = io.ReadFull(r, data)
	if err != nil {
		return nil, fmt.Errorf("Unable to read full data of bulk string::: %w", err)
	}

	// Consume trailing CRLF so subsequent connection commands start "clean"
	r.ReadString('\n')

	return &Value{
		Type:   BulkString,
		Length: length,
		String: string(data),
	}, nil
}

func parseSimpleString(r *bufio.Reader) (*Value, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("Unable to read simple string line::: %w", err)
	}
	return &Value{
		Type:   SimpleString,
		String: strings.TrimSpace(line),
	}, nil
}
