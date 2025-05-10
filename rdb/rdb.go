package rdb

import (
	"bufio"
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
	// NOTE: can use `readHeader` returned magic string and version to validate
	_, err = readHeader(r)
	if err != nil {
		return err
	}

	// 2. Metadata
	err = readMetadata(r)
	if err != nil {
		return err
	}

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
	// NOTE: probably should just ReadByte since it should be `0xfa` anyway
	b, err := r.ReadBytes(0xfa)
	if err != nil {
		return fmt.Errorf("%s file read: metadata: %w", ErrLoadPrefix, err)
	}

	fmt.Printf("\ntype: %T\nmetadata: %q\n", b, b)
	return nil
}
