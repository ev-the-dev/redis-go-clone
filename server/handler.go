package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/ev-the-dev/redis-go-clone/resp"
)

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
			s.handleEchoCommand(conn, val)
		case "GET":
			s.handleGetCommand(conn, val)
		case "SET":
			s.handleSetCommand(conn, val)
		default:
			conn.Write([]byte(resp.EncodeSimpleErr("Unknown command")))
		}
	}
}

func (s *Server) handleEchoCommand(conn net.Conn, val *resp.Value) {
	if len(val.Array) != 2 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `ECHO` command")))
		return
	}

	argVal := val.Array[1]
	if argVal.Type != resp.BulkString {
		conn.Write([]byte(resp.EncodeSimpleErr("Argument to `ECHO` command must be bulk string type")))
		return
	}

	conn.Write([]byte(resp.EncodeBulkString(argVal.String)))
}

func (s *Server) handleGetCommand(conn net.Conn, _ *resp.Value) {
	conn.Write([]byte(resp.EncodeSimpleErr("Not yet implemented.")))
	return
}

func (s *Server) handleSetCommand(conn net.Conn, val *resp.Value) {
	if len(val.Array) <= 2 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `SET` command")))
		return
	}
	// 	keyVal := val.Array[1]
	// 	valVal := val.Array[2]

	conn.Write([]byte(resp.EncodeSimpleString("OK")))
	return
}
