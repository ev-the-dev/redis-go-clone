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

	bufR := bufio.NewReaderSize(file, 32*1024)

	header := make([]byte, 9)

	if _, err := io.ReadFull(bufR, header); err != nil {
		return fmt.Errorf("%s file read: %w", ErrLoadPrefix, err)
	}

	fmt.Printf("header: %s\n", header)

	return nil
}
