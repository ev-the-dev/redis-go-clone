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
		var a, v string
		argPair := strings.Split(args[i], "=")
		a = argPair[0]
		if len(argPair) == 2 {
			v = argPair[1]
		}

		switch strings.ToLower(a) {
		case "--dir":
			if v != "" {
				cfg.Dir = v
			} else {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("%s parse: --dir requires argument", ErrMainArg)
				}
				cfg.Dir = args[i+1]
				i++
			}
		case "--dbfilename":
			if v != "" {
				cfg.DBFilename = v
			} else {
				if i+1 >= len(args) {
					return nil, fmt.Errorf("%s parse: --dbfilename requires argument", ErrMainArg)
				}
				cfg.DBFilename = args[i+1]
				i++
			}
		}
	}
	return cfg, nil
}
