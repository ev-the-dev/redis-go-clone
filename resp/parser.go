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
		return &Value{}, fmt.Errorf("%s first byte: %w", ErrProtocolPrefix, err)
	}

	switch firstByte {
	case '+': // SimpleString
		return parseSimpleString(r)
	case '$': // BulkString
		return parseBulkString(r)
	case '*': // Array
		return parseArray(r)
	default:
		return nil, fmt.Errorf("%s unknown type: %q", ErrProtocolPrefix, firstByte)
	}
}

func parseArray(r *bufio.Reader) (*Value, error) {
	arrLen, err := r.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("%s array: length: %w", ErrParsePrefix, err)
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
			return nil, fmt.Errorf("%s array: recursion: %w", ErrParsePrefix, err)
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
		return nil, fmt.Errorf("%s bulk string: length: %w", ErrParsePrefix, err)
	}
	strLen = strings.TrimSpace(strLen)
	length, err := strconv.Atoi(strLen)
	if err != nil {
		return nil, fmt.Errorf("%s bulk string: length to int: %w", ErrParsePrefix, err)
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
		return nil, fmt.Errorf("%s bulk string: read full: %w", ErrParsePrefix, err)
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
		return nil, fmt.Errorf("%s simple string: read: %w", ErrParsePrefix, err)
	}
	return &Value{
		Type:   SimpleString,
		String: strings.TrimSpace(line),
	}, nil
}
