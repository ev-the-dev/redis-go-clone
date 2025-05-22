package rdb

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/ev-the-dev/redis-go-clone/store"
)

func Load(path string, s *store.Store) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("%s file missing: %s\n", ErrLoadPrefix, path)
			return nil
		}
		return fmt.Errorf("%s file load: %w", ErrLoadPrefix, err)
	}
	defer file.Close()

	r := bufio.NewReaderSize(file, 64*1024)

	// 1. Header
	// NOTE: can use `readHeader`'s returned magic string and version to validate
	_, err = readHeader(r)
	if err != nil {
		return err
	}

	// 2. Metadata
	err = readMetadata(r)
	if err != nil {
		return err
	}

	// 3. Database Selections

	// 4. Footer

	return nil
}

func readHeader(r io.Reader) (string, error) {
	header := make([]byte, 9)

	if _, err := io.ReadFull(r, header); err != nil {
		return "", fmt.Errorf("%s file read: header: %w", ErrLoadPrefix, err)
	}

	return string(header), nil
}

// TODO: create parser for length encoding
// after each delim for key-value pair, first 2 bits are length encoding
// i.e.
//	00 - next 6 bits == length
//	01 - next 14 bits == length
//	10 - discard next 6 bits, next 4 bytes == length
//	11 - special format encoding, remaining 6 bits indicate format

func readMetadata(r *bufio.Reader) error {
	// 1. Read 0xFA OP Code
	b, err := r.ReadByte()
	if err != nil {
		return fmt.Errorf("%s file read: metadata: first byte: %w", ErrLoadPrefix, err)
	}

	if b != 0xFA {
		return fmt.Errorf("%s file read: metadata: first byte: expected 0xFA but got 0x%X", ErrLoadPrefix, b)
	}

	fmt.Printf("\ntype: %T\nmetadata: %X\n", b, b)

	// 2. Read Length-Encoded Descriptor

	return nil
}

func parseLengthEncoded(r *bufio.Reader) (uint16, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("%s first byte: %w", ErrLengthEncodePrefix, err)
	}

	switch prefix := b >> 6; prefix {
	case 0: // 00xxxxxx
		// Grab next 6 bits for the total length
		l := b & 0x3F
		return uint16(l), nil
	case 1: // 01xxxxxx
		// Grab next 6 bits, plus read a byte and add those 8 bits to get the total length
		l := b & 0x3F
		b, err = r.ReadByte()
		if err != nil {
			return 0, fmt.Errorf("%s second byte: %w", ErrLengthEncodePrefix, err)
		}

	case 2: // 10xxxxxx
	case 3: // 11xxxxxx
	default:
		return 0, fmt.Errorf("%s impossible significant bits: %w", ErrLengthEncodePrefix, err)
	}
}
