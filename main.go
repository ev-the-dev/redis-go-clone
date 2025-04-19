package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ev-the-dev/redis-go-clone/config"
	"github.com/ev-the-dev/redis-go-clone/server"
)

const ErrMainArg = "main: arg:"

func main() {
	args := os.Args[1:]
	cfg, err := parseArgs(args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	s := server.New(cfg)
	s.Start()
}

// TODO: validate that the args have present and valid values (directory and filename)
func parseArgs(args []string) (*config.Config, error) {
	cfg := config.New()
	for i := 0; i < len(args); i++ {
		switch strings.ToLower(args[i]) {
		case "--dir":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%s parse: --dir requires argument", ErrMainArg)
			}
			cfg.Dir = args[i+1]
			i++
		case "--dbfilename":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("%s parse: --dbfilename requires argument", ErrMainArg)
			}
			cfg.DBFilename = args[i+1]
			i++
		}
	}
	return cfg, nil
}
