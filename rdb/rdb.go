package rdb

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

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
	err = readDatabases(r, s)
	if err != nil {
		return err
	}

	// 4. Footer
	err = readFooter(r)
	if err != nil {
		return err
	}

	// 5. Ensure EOF
	if b, err := r.ReadByte(); err != nil && err != io.EOF {
		return fmt.Errorf("%s expected EOF: got 0x%X", ErrLoadPrefix, b)
	}

	return nil
}

type LocalEntry struct {
	Expire  time.Time
	Key     any
	KeyType ValueType
	Val     any
	ValType ValueType
}

func readHeader(r io.Reader) (string, error) {
	header := make([]byte, 9)

	if _, err := io.ReadFull(r, header); err != nil {
		return "", fmt.Errorf("%s magic string: %w", ErrReadHeader, err)
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
			r.UnreadByte()
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
			return fmt.Errorf("%s key type: %w", ErrReadMetadata, err)
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
		fmt.Printf("Metadata Entry: %+v\n", lE)
	}
}

func readDatabases(r *bufio.Reader, s *store.Store) error {
	// 1a. Read 0xFE OP Code
	b, err := r.ReadByte()
	if err != nil {
		return fmt.Errorf("%s 0xFE byte: %w", ErrReadDatabase, err)
	}

	if b != 0xFE {
		return fmt.Errorf("%s 0xFE byte: got 0x%X", ErrReadDatabase, b)
	}

	// 1b. Read DB Number
	dbNum, err := r.ReadByte()
	if err != nil {
		return fmt.Errorf("%s DB number: %w", ErrReadDatabase, err)
	}

	fmt.Printf("Database Number (%d)\n", uint32(dbNum))

	// 2a. Read 0xFB OP Code
	b, err = r.ReadByte()
	if err != nil {
		return fmt.Errorf("%s 0xFB byte: %w", ErrReadDatabase, err)
	}

	if b != 0xFB {
		return fmt.Errorf("%s 0xFB byte: got 0x%X", ErrReadDatabase, b)
	}

	// 2b. Read Size of Hash & Expire Table
	hashBytes := make([]byte, 2)
	if _, err := io.ReadFull(r, hashBytes); err != nil {
		return fmt.Errorf("%s 0xFB byte: hash and expire table: %w", ErrReadDatabase, err)
	}

	fmt.Printf("Hash Table Size (%d)\nExpire Table Size (%d)\n", uint32(hashBytes[0]), uint32(hashBytes[1]))

	// 3. Read Main DB Data
	for {
		lE := &LocalEntry{}
		b, err = r.ReadByte()
		if err != nil {
			return fmt.Errorf("%s record first byte: %w", ErrReadDatabase, err)
		}

		// 3a. Read Optional Expiry or Encounter New DB or EOF
		switch b {
		case 0xFD: // Unix Seconds Timestamp, read 4 bytes, little-endian
			timeBytes := make([]byte, 4)
			if _, err := io.ReadFull(r, timeBytes); err != nil {
				return fmt.Errorf("%s 0xFD byte: %w", ErrReadDatabase, err)
			}
			lE.Expire = time.Unix(int64(binary.LittleEndian.Uint64(timeBytes)), 0)
		case 0xFC: // Unix Milliseconds Timestamp, read 8 bytes, little-endian
			timeBytes := make([]byte, 8)
			if _, err := io.ReadFull(r, timeBytes); err != nil {
				return fmt.Errorf("%s 0xFC byte: %w", ErrReadDatabase, err)
			}
			lE.Expire = time.UnixMilli(int64(binary.LittleEndian.Uint64(timeBytes)))
		case 0xFE: // Old DB Ends, New Begins
			r.UnreadByte()
			return readDatabases(r, s)
		case 0xFF: // End of RDB File
			r.UnreadByte()
			return nil

		default: // Unread byte and handle afterwards
			r.UnreadByte()
		}

		// 3b. Read ValueType
		vt, err := r.ReadByte()
		if err != nil {
			return fmt.Errorf("%s ValueType: %w", ErrReadDatabase, err)
		}

		lE.ValType = ValueType(vt)

		// 3c. Read String-Encoded Key
		pL, err := parseLengthEncoded(r, StringEncoded)
		if err != nil {
			return fmt.Errorf("%s %w", ErrReadDatabase, err)
		}

		lE.KeyType = pL.ValType

		pSD, err := parseStringData(r, pL)
		if err != nil {
			return fmt.Errorf("%s %w", ErrReadDatabase, err)
		}

		lE.Key = pSD

		// 3d. Read ValueType Value
		pL, err = parseLengthEncoded(r, lE.ValType)
		pD, err := parseData(r, pL)
		if err != nil {
			return fmt.Errorf("%s %w", ErrReadDatabase, err)
		}

		lE.Val = pD

		// 4. Store Key:Value to Store
		fmt.Printf("Database Entry: %+v\n", lE)
	}
}

func readFooter(r *bufio.Reader) error {
	// 1. Read 0xFF OP Code
	b, err := r.ReadByte()
	if err != nil {
		return fmt.Errorf("%s 0xFF byte: %w", ErrReadFooter, err)
	}

	if b != 0xFF {
		return fmt.Errorf("%s 0xFF byte: got 0x%X", ErrReadFooter, b)
	}

	fmt.Printf("\nReached the end of the RDB File!\n")

	// 2. Read File Checksum
	chsumBytes := make([]byte, 8)
	if _, err := io.ReadFull(r, chsumBytes); err != nil {
		return fmt.Errorf("%s checksum: %w", ErrReadFooter, err)
	}

	chsum := binary.LittleEndian.Uint64(chsumBytes)

	fmt.Printf("Checksum of file: %d\n", chsum)

	return nil
}
