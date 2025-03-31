package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/ev-the-dev/redis-go-clone/resp"
	"github.com/ev-the-dev/redis-go-clone/store"
)

func main() {
	s := NewServer()
	s.Start()
}

type Server struct {
	store *store.Store
}

func NewServer() *Server {
	return &Server{
		store: store.New(),
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		val, err := resp.Parse(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("Client disconnected.")
				return
			}
			fmt.Printf("%s cmd: %v\n", ErrCmdPrefix, err)
			continue
		}

		if val.Type != resp.Array || len(val.Array) <= 0 {
			conn.Write([]byte(resp.EncodeSimpleErr("Expected command array")))
			continue
		}

		cmdVal := val.Array[0]
		if cmdVal.Type != resp.BulkString {
			conn.Write([]byte(resp.EncodeSimpleErr("Command must be bulk string type")))
			continue
		}

		switch strings.ToUpper(cmdVal.String) {
		case "PING":
			conn.Write([]byte("+PONG\r\n"))
		case "ECHO":
			if len(val.Array) != 2 {
				conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `ECHO` command")))
				continue
			}

			argVal := val.Array[1]
			if argVal.Type != resp.BulkString {
				conn.Write([]byte(resp.EncodeSimpleErr("Argument to `ECHO` command must be bulk string type")))
				continue
			}

			conn.Write([]byte(resp.EncodeBulkString(argVal.String)))
		// case "GET":
		// case "SET":
		// 	if len(val.Array) <= 2 {
		// 		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `SET` command")))
		// 		continue
		// 	}
		//
		// 	keyVal := val.Array[1]
		// 	valVal := val.Array[2]

		default:
			conn.Write([]byte(resp.EncodeSimpleErr("Unknown command")))
		}
	}
}

func (s *Server) Start() {
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Printf("%s port: %v\n", ErrConnPrefix, err)
		os.Exit(1)
	}

	fmt.Println("Listening on port: 6379")

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Printf("%s client: %v\n", ErrConnPrefix, err.Error())
			continue
		}

		go s.handleConnection(conn)
	}
}
