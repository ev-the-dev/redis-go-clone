package rdb

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strconv"

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

type LocalEntry struct {
	Key     any
	KeyType ValueType
	Val     any
	ValType ValueType
}

func readHeader(r io.Reader) (string, error) {
	header := make([]byte, 9)

	if _, err := io.ReadFull(r, header); err != nil {
		return "", fmt.Errorf("%s file read: header: %w", ErrLoadPrefix, err)
	}

	return string(header), nil
}

func readMetadata(r *bufio.Reader) error {
	for {
		// 1. Read 0xFA OP Code
		b, err := r.ReadByte()
		if err != nil {
			return fmt.Errorf("%s file read: metadata: first byte: %w", ErrLoadPrefix, err)
		}

		// If a DB marker is found, then we've finished reading from the metadata section
		if b == 0xFE {
			return nil
		}

		if b != 0xFA {
			return fmt.Errorf("%s file read: metadata: first byte: expected 0xFA but got 0x%X", ErrLoadPrefix, b)
		}

		lE := &LocalEntry{}
		// 2. Begin Read Key
		// 2a. Read Length-Encoded Descriptor
		pL, err := parseLengthEncoded(r, StringEncoded)
		if err != nil {
			return fmt.Errorf("%s key: %w", ErrReadMetadata, err)
		}

		lE.KeyType = pL.ValType

		// 2b. Read Value of Key
		pD, err := parseData(r, pL)
		if err != nil {
			return fmt.Errorf("%s key: %w", ErrReadMetadata, err)
		}

		lE.Key = pD

		// 3. Begin Read Value
		// 3a. Read Length-Encoded Descriptor
		pL, err = parseLengthEncoded(r, StringEncoded)
		if err != nil {
			return fmt.Errorf("%s value: %w", ErrReadMetadata, err)
		}

		lE.ValType = pL.ValType

		// 3b. Read Value of Value
		pD, err = parseData(r, pL)
		if err != nil {
			return fmt.Errorf("%s value: %w", ErrReadMetadata, err)
		}

		lE.Val = pD

		// 4. Store Key:Value to Store
		fmt.Printf("LocalEntry: %+v\n", lE)
	}
}

func parseData(r *bufio.Reader, pL *ParseLength) (any, error) {
	switch pL.ValType {
	case StringEncoded:
		return parseStringData(r, pL)
	case ListEncoded:
		return parseListData(r, pL)
	default:
		return nil, fmt.Errorf("%s unsupported ValueType: %d", ErrParseDataPrefix, pL.ValType)
	}
}

func parseListData(r *bufio.Reader, pL *ParseLength) ([]string, error) {
	list := make([]string, pL.Length)
	for range pL.Length {
		p, err := parseLengthEncoded(r, StringEncoded)
		if err != nil {
			return nil, fmt.Errorf("%s list: %w", ErrParseDataPrefix, err)
		}

		s, err := parseStringData(r, p)
		if err != nil {
			return nil, fmt.Errorf("%s list: %w", ErrParseDataPrefix, err)
		}

		list = append(list, s)
	}

	return list, nil
}

func parseStringData(r *bufio.Reader, pL *ParseLength) (string, error) {
	if pL.IsSpecial {
		switch pL.SpecialType {
		case SpecialInt8:
			b, err := r.ReadByte()
			if err != nil {
				return "", fmt.Errorf("%s string: special int8: %w", ErrParseDataPrefix, err)
			}
			return strconv.Itoa(int(int8(b))), nil
		case SpecialInt16:
			b := make([]byte, 2)
			if _, err := io.ReadFull(r, b); err != nil {
				return "", fmt.Errorf("%s string: special int16: %w", ErrParseDataPrefix, err)
			}
			num := int16(binary.LittleEndian.Uint16(b))
			return strconv.Itoa(int(num)), nil
		case SpecialInt32:
			b := make([]byte, 4)
			if _, err := io.ReadFull(r, b); err != nil {
				return "", fmt.Errorf("%s string: special int32: %w", ErrParseDataPrefix, err)
			}
			num := int32(binary.LittleEndian.Uint32(b))
			return strconv.Itoa(int(num)), nil
		case SpecialLZF:
			return "", fmt.Errorf("% string: special LZF: NOT IMPLEMENTED YET", ErrParseDataPrefix)
		default:
			return "", fmt.Errorf("% string: special: unsupported special type: %d", ErrParseDataPrefix, pL.SpecialType)
		}
	}

	b := make([]byte, pL.Length)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", fmt.Errorf("%s string: read full: %w", ErrParseDataPrefix, err)
	}

	fmt.Printf("STRING::: %v\n", string(b))
	return string(b), nil
}

type ParseLength struct {
	IsSpecial   bool
	Length      uint32
	SpecialType SpecialLengthType
	ValType     ValueType
}

func parseLengthEncoded(r *bufio.Reader, vt ValueType) (*ParseLength, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("%s first byte: %w", ErrLengthEncodePrefix, err)
	}

	switch prefix := b >> 6; prefix {
	case 0: // 00xxxxxx
		// Grab next 6 bits for the total length
		l := b & 0x3F
		return &ParseLength{
			IsSpecial: false,
			Length:    uint32(l),
			ValType:   vt,
		}, nil
	case 1: // 01xxxxxx
		// Grab next 6 bits, plus read a byte and add those 8 bits to get the total length
		l1 := b & 0x3F
		b, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("%s case 1: %w", ErrLengthEncodePrefix, err)
		}
		// Can't do bit operations on this before alloc more mem. Each byte read from `ReadByte` only allocates 8 bits. We need at least 14 as per the protocol for this case. Need to ensure enough mem to shift by 8 bits so that way we can use the OR operator to "concat" the second byte's 8 bits onto the end.
		lengthTotal := (uint32(l1)<<8 | uint32(b))
		return &ParseLength{
			IsSpecial: false,
			Length:    lengthTotal,
			ValType:   vt,
		}, nil
	case 2: // 10xxxxxx
		// Discard remaining 6 bits, then use the next 4 bytes as the total length
		l := make([]byte, 4)
		if _, err := io.ReadFull(r, l); err != nil {
			return nil, fmt.Errorf("%s case 2: %w", ErrLengthEncodePrefix, err)
		}
		return &ParseLength{
			IsSpecial: false,
			Length:    binary.BigEndian.Uint32(l),
			ValType:   vt,
		}, nil
	case 3: // 11xxxxxx
		// Special format -- next 6 bits describe the format
		specialType := b & 0x3F
		return parseLengthEncodedSpecialFormat(specialType)
	default:
		return nil, fmt.Errorf("%s impossible significant bits", ErrLengthEncodePrefix)
	}
}

func parseLengthEncodedSpecialFormat(bits byte) (*ParseLength, error) {
	pL := &ParseLength{
		IsSpecial: true,
		ValType:   StringEncoded,
	}
	switch bits {
	case 0: // 8-bit integer, read next byte for value
		pL.Length = 1
		pL.SpecialType = SpecialInt8
		return pL, nil
	case 1: // 16-bit integer, read next 2 bytes for value
		pL.Length = 2
		pL.SpecialType = SpecialInt16
		return pL, nil
	case 2: // 32-bit integer, read next 4 bytes for value
		pL.Length = 4
		pL.SpecialType = SpecialInt32
		return pL, nil
	case 3: // LZF compressed string
		pL.SpecialType = SpecialLZF
		return pL, nil
	default:
		return nil, fmt.Errorf("%s impossible remaining bits value", ErrSpecialLengthEncodePrefix)
	}
}
