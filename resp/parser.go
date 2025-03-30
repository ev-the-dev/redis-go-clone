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
	case '*': // Array
		return parseArray(r)
	default:
		return nil, fmt.Errorf("Unknown RESP type::: %c", firstByte)
	}
}

func parseArray(r *bufio.Reader) (*Value, error) {
	arrLen, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("Unable to read array length::: %w", err)
	}
	arrLen = strings.TrimSpace(arrLen)
	length, err := strconv.Atoi(arrLen)

	// NOTE: not entirely sure the distinction yet between
	// a null array and zero-lengthed array from a RESP
	// perspective.
	if length <= 0 {
		return &Value{
			Type:   Array,
			Length: length,
		}, nil
	}

	arr := make([]*Value, 0, length)
	for range length {
		val, err := Parse(r)
		if err != nil {
			return nil, fmt.Errorf("Encountered error recursing over array::: %w", err)
		}

		arr = append(arr, val)
	}

	return &Value{
		Type:   Array,
		Array:  arr,
		Length: length,
	}, nil
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
