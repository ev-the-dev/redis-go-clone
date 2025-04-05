package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	"github.com/ev-the-dev/redis-go-clone/resp"
)

func (s *Server) handleConfigCommand(conn net.Conn, msg *resp.Message) {
	// NOTE: if I need support just the `CONFIG` command this needs to change
	if len(msg.Array) < 3 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `CONFIG *` command")))
		return
	}

	subCmd := msg.Array[1]
	switch strings.ToUpper(subCmd.String) {
	case "GET":
		s.handleConfigGetCommand(conn, msg)
	default:
		conn.Write([]byte(resp.EncodeSimpleErr("Unknown CONFIG subcommand")))
	}
}

// TODO: Need to "build" the bulkstring for all params/indices instead of writing
// on a per case basis like below
func (s *Server) handleConfigGetCommand(conn net.Conn, msg *resp.Message) {
	// Starting at 2 because `CONFIG` is 0 and `GET` is 1
	for i := 2; i < len(msg.Array); i++ {
		m := msg.Array[i]
		fmt.Printf("M(%d): %+v\n", i, m)
		// TODO: support glob pattern matching
		switch strings.ToLower(m.String) {
		case "dir":
			conn.Write([]byte(resp.EncodeArray(
				resp.EncodeBulkString("dir"),
				resp.EncodeBulkString(s.config.Dir),
			)))
		case "dbfilename":
			conn.Write([]byte(resp.EncodeArray(
				resp.EncodeBulkString("dbfilename"),
				resp.EncodeBulkString(s.config.DBFilename),
			)))
		default:
			conn.Write([]byte(resp.EncodeSimpleErr("Unrecognized config key")))
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		msg, err := resp.Parse(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("Client disconnected.")
				return
			}
			log.Printf("%s %v\n", ErrCmdPrefix, err)
			continue
		}

		if msg.Type != resp.Array || len(msg.Array) <= 0 {
			conn.Write([]byte(resp.EncodeSimpleErr("Expected command array")))
			continue
		}

		cmdMsg := msg.Array[0]
		if cmdMsg.Type != resp.BulkString {
			conn.Write([]byte(resp.EncodeSimpleErr("Command must be bulk string type")))
			continue
		}

		switch strings.ToUpper(cmdMsg.String) {
		case "PING":
			conn.Write([]byte("+PONG\r\n"))
		case "CONFIG":
			s.handleConfigCommand(conn, msg)
		case "ECHO":
			s.handleEchoCommand(conn, msg)
		case "GET":
			s.handleGetCommand(conn, msg)
		case "SET":
			s.handleSetCommand(conn, msg)
		default:
			conn.Write([]byte(resp.EncodeSimpleErr("Unknown command")))
		}
	}
}

func (s *Server) handleEchoCommand(conn net.Conn, msg *resp.Message) {
	if len(msg.Array) != 2 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `ECHO` command")))
		return
	}

	argVal := msg.Array[1]
	if argVal.Type != resp.BulkString {
		conn.Write([]byte(resp.EncodeSimpleErr("Argument to `ECHO` command must be bulk string type")))
		return
	}

	conn.Write([]byte(resp.EncodeBulkString(argVal.String)))
}

func (s *Server) handleGetCommand(conn net.Conn, msg *resp.Message) {
	if len(msg.Array) <= 1 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `GET` command")))
		return
	}

	keyMsg := msg.Array[1]

	key, err := keyMsg.GetString()
	if err != nil {
		log.Printf("%s: GET: invalid key: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Invalid key type for `GET` command")))
		return
	}

	record, exists := s.store.Get(key)
	if !exists {
		conn.Write([]byte(resp.EncodeNullBulkString()))
		return
	}

	conn.Write([]byte(resp.EncodeBulkString(record.Value)))
	return
}

func (s *Server) handleSetCommand(conn net.Conn, msg *resp.Message) {
	if len(msg.Array) <= 2 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `SET` command")))
		return
	}

	keyMsg := msg.Array[1]
	valMsg := msg.Array[2]

	key, err := keyMsg.GetString()
	if err != nil {
		log.Printf("%s: SET: invalid key: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Invalid key type for `SET` command")))
		return
	}

	val, err := valMsg.GetString()
	if err != nil {
		log.Printf("%s: SET: invalid value: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Invalid value type for `SET` command")))
		return
	}

	opts := &SetOptions{}
	if len(msg.Array) > 3 {
		opts, err = parseSETOptions(msg.Array[3:])
		if err != nil {
			log.Println(err)
			conn.Write([]byte(resp.EncodeSimpleErr("Invalid data type for `SET` command option")))
			return
		}
	}

	if opts.KEEPTTL {
		if record, exists := s.store.Get(key); exists {
			opts.Expiry = record.ExpiresAt
		}
	}

	s.store.Set(key, val, opts.Expiry)

	if opts.GET {
		conn.Write([]byte(resp.EncodeBulkString(val)))
	} else {
		conn.Write([]byte(resp.EncodeSimpleString("OK")))
	}
	return
}
