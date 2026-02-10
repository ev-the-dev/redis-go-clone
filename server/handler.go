package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/ev-the-dev/redis-go-clone/resp"
	"github.com/ev-the-dev/redis-go-clone/store"
)

func (s *Server) handleBLPOPCommand(conn net.Conn, msg *resp.Message) {
	if len(msg.Array) < 2 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `BLPOP` command")))
		return
	}

	keyMsgs := msg.Array[1 : len(msg.Array)-1]
	timeoutMsg := msg.Array[len(msg.Array)-1]

	timeout, err := timeoutMsg.ConvInt()
	if err != nil {
		log.Printf("%s: BLPOP: invalid timeout: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Timeout arg for `BLPOP` must be greater than or equal to 0")))
		return
	}
	if timeout == 0 {
		timeout = 1_000_000
	}

	emptyKeys := make([]string, 0, len(keyMsgs)/2)
	for i, km := range keyMsgs {
		key, err := km.ConvStr()
		if err != nil {
			log.Printf("%s: BLPOP: invalid key at pos (%d): %v", ErrCmdPrefix, i, err)
			conn.Write([]byte(resp.EncodeSimpleErr("Invalid key type for `BLPOP` command")))
			return
		}

		record, exists := s.store.Get(key)
		if !exists || (record.Type == resp.Array && len(record.Value.([]*store.Record)) == 0) {
			emptyKeys = append(emptyKeys, key)
			continue
		}

		if record.Type != resp.Array {
			log.Printf("%s: BLPOP: invalid type from key (%s): %s", ErrCmdPrefix, key, record.Type.String())
			conn.Write([]byte(resp.EncodeSimpleErr(fmt.Sprintf("Provided `BLPOP` Key (%s) produced non array/list type", key))))
			return
		}

		list := record.Value.([]*store.Record)
		val := list[0]
		record.Value = list[1:]

		s.store.Set(key, record)

		toResp, err := toRESPString(val)
		if err != nil {
			log.Printf("%s: BLPOP: to resp string: %v", ErrCmdPrefix, err)
			conn.Write([]byte(resp.EncodeSimpleErr("Unable to output popped value")))
			return
		}

		conn.Write([]byte(resp.EncodeArray(2, []string{resp.EncodeBulkString(key), toResp}...)))
		return
	}

	/*** BLOCKING BEGINS ***/
	bc := &BlockedClient{
		conn:    conn,
		replyCh: make(chan *BlockedClientChanResp, 1),
		subs:    emptyKeys,
	}

	s.blockingManager.RegisterClient(bc)

	select {
	case res, ok := <-bc.replyCh:
		if !ok {
			// NOTE: not entirely sure what to do here
		}
		if res.rec.Type != resp.Array {
			log.Printf("%s: BLPOP: blocking: invalid type from key (%s): %s", ErrCmdPrefix, res.key, res.rec.Type.String())
			conn.Write([]byte(resp.EncodeSimpleErr(fmt.Sprintf("Provided `BLPOP` Key (%s) produced non array/list type", res.key))))
			return
		}

		list := res.rec.Value.([]*store.Record)
		val := list[0]
		res.rec.Value = list[1:]

		s.store.Set(res.key, res.rec)

		if res.rec.Type == resp.Array && len(res.rec.Value.([]*store.Record)) != 0 {
			s.blockingManager.NotifyWatchers(res.key, res.rec)
		}

		toResp, err := toRESPString(val)
		if err != nil {
			log.Printf("%s: BLPOP: blocking: to resp string: %v", ErrCmdPrefix, err)
			conn.Write([]byte(resp.EncodeSimpleErr("Unable to output blocked popped value")))
			return
		}
		conn.Write([]byte(resp.EncodeArray(2, []string{resp.EncodeBulkString(res.key), toResp}...)))
	case <-time.After(time.Duration(timeout) * time.Second):
		conn.Write([]byte(resp.EncodeNullArray()))
		s.blockingManager.UnregisterClient(bc)
	}
}

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

func (s *Server) handleConfigGetCommand(conn net.Conn, msg *resp.Message) {
	result := make([]string, 0, len(msg.Array)*2)
	// Starting at 2 because `CONFIG` is 0 and `GET` is 1
	for i := 2; i < len(msg.Array); i++ {
		m := msg.Array[i]
		// TODO: support glob pattern matching
		switch strings.ToLower(m.String) {
		case "dir":
			result = append(result, resp.EncodeBulkString("dir"), resp.EncodeBulkString(s.config.Dir))
		case "dbfilename":
			result = append(result, resp.EncodeBulkString("dbfilename"), resp.EncodeBulkString(s.config.DBFilename))
		default:
			conn.Write([]byte(resp.EncodeSimpleErr("Unrecognized config key")))
		}
	}

	conn.Write([]byte(resp.EncodeArray(len(result), result...)))
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		// Parse RESP command
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

		switch CmdName(strings.ToUpper(cmdMsg.String)) {
		case PING:
			conn.Write([]byte("+PONG\r\n"))
		case BLPOP:
			s.handleBLPOPCommand(conn, msg)
		case CONFIG:
			s.handleConfigCommand(conn, msg)
		case ECHO:
			s.handleEchoCommand(conn, msg)
		case GET:
			s.handleGetCommand(conn, msg)
		case KEYS:
			s.handleKeysCommand(conn, msg)
		case LLEN:
			s.handleLlenCommand(conn, msg)
		case LPOP:
			s.handleLpopCommand(conn, msg)
		case LPUSH:
			s.handleLpushCommand(conn, msg)
		case LRANGE:
			s.handleLrangeCommand(conn, msg)
		case RPUSH:
			s.handleRpushCommand(conn, msg)
		case SET:
			s.handleSetCommand(conn, msg)
		case TYPE:
			s.handleTypeCommand(conn, msg)
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

	key, err := keyMsg.ConvStr()
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

	respVal, err := toRESPString(record)
	if err != nil {
		log.Printf("%s: GET: resp string: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Unable to retrieve SET value")))
	}

	conn.Write([]byte(respVal))
}

func (s *Server) handleKeysCommand(conn net.Conn, msg *resp.Message) {
	if len(msg.Array) != 2 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `KEYS` command")))
		return
	}

	patternMsg := msg.Array[1]
	if patternMsg.Type != resp.SimpleString && patternMsg.Type != resp.BulkString {
		conn.Write([]byte(resp.EncodeSimpleErr("`KEYS` pattern must be a string, i.e. '*'")))
		return
	}

	pattern := patternMsg.String

	result := make([]string, 0, len(msg.Array)*2)
	for _, k := range s.store.Keys() {
		match, err := filepath.Match(pattern, k)
		if err != nil {
			conn.Write([]byte(resp.EncodeSimpleErr("Error matching pattern for `KEYS` command")))
		}

		if match {
			result = append(result, resp.EncodeBulkString(k))
		}
	}

	conn.Write([]byte(resp.EncodeArray(len(result), result...)))
}

func (s *Server) handleLlenCommand(conn net.Conn, msg *resp.Message) {
	if len(msg.Array) != 2 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `LLEN` command")))
		return
	}

	keyMsg := msg.Array[1]

	key, err := keyMsg.ConvStr()
	if err != nil {
		log.Printf("%s: LLEN: invalid key name: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Invalid key name type for `LLEN` command")))
		return
	}

	record, exists := s.store.Get(key)
	if !exists {
		conn.Write([]byte(resp.EncodeInteger(0)))
		return
	}

	var length int
	switch record.Type {
	case resp.Array, resp.Sets:
		length = len(record.Value.([]*store.Record))
	case resp.Maps:
		length = len(record.Value.(map[string]*store.Record))
	default:
		log.Printf("%s: LLEN: invalid type: %s", ErrCmdPrefix, record.Type.String())
		conn.Write([]byte(resp.EncodeSimpleErr("Provided `LLEN` Key produced non-keyed type")))
		return
	}

	conn.Write([]byte(resp.EncodeInteger(length)))
}

func (s *Server) handleLpopCommand(conn net.Conn, msg *resp.Message) {
	if len(msg.Array) < 2 || len(msg.Array) > 3 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `LPOP` command")))
		return
	}

	keyMsg := msg.Array[1]

	key, err := keyMsg.ConvStr()
	if err != nil {
		log.Printf("%s: LPOP: invalid key name: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Invalid key name type for `LPOP` command")))
		return
	}

	record, exists := s.store.Get(key)
	if !exists {
		log.Printf("%s: LPOP: does not exists: %s", ErrCmdPrefix, key)
		conn.Write([]byte(resp.EncodeNulls()))
		return
	}

	if record.Type != resp.Array {
		log.Printf("%s: LPOP: invalid type: %s", ErrCmdPrefix, record.Type.String())
		conn.Write([]byte(resp.EncodeSimpleErr("Provided `LPOP` Key produced non array/list type")))
		return
	}

	list := record.Value.([]*store.Record)

	if len(list) == 0 {
		conn.Write([]byte(resp.EncodeNulls()))
		return
	}

	count := 1
	if len(msg.Array) == 3 {
		countMsg := msg.Array[2]
		count, err = countMsg.ConvInt()
		if err != nil {
			log.Printf("%s: LPOP: count parse: %v", ErrCmdPrefix, err)
			conn.Write([]byte(resp.EncodeSimpleErr("Unable to parse `LPOP` [count] arg")))
			return
		}

		if count < 0 {
			conn.Write([]byte(resp.EncodeSimpleErr("`LPOP` [count] arg must be positive")))
			return
		}
	}

	count = min(len(list), count)

	poppedSlice := list[0:count]
	record.Value = list[count:]

	s.store.Set(key, record)

	toResp, err := toBulkRESPString(poppedSlice)
	if err != nil {
		log.Printf("%s: LPOP: to resp string: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Unable to output popped array")))
		return
	}

	conn.Write([]byte(resp.EncodeArray(len(toResp), toResp...)))
}

func (s *Server) handleLpushCommand(conn net.Conn, msg *resp.Message) {
	if len(msg.Array) <= 2 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `LPUSH` command")))
		return
	}

	keyMsg := msg.Array[1]
	valMsgs := msg.Array[2:]

	key, err := keyMsg.ConvStr()
	if err != nil {
		log.Printf("%s: LPUSH: invalid key name: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Invalid key name type for `LPUSH` command")))
		return
	}

	record, exists := s.store.Get(key)
	if !exists {
		record = &store.Record{
			Type:  resp.Array,
			Value: make([]*store.Record, 0, len(valMsgs)),
		}
	}

	if record.Type != resp.Array {
		log.Printf("%s: LPUSH: invalid type: %s", ErrCmdPrefix, record.Type.String())
		conn.Write([]byte(resp.EncodeSimpleErr("Provided `LPUSH` Key produced non array/list type")))
		return
	}

	// NOTE: Can improve perf by iterating over half the slice/array instead and swapping
	// values with the current index and the 0th + (len - i)th index. Though in practical
	// situations this will probably be negligible.
	newVals := make([]*store.Record, len(valMsgs))
	for i := len(valMsgs) - 1; i >= 0; i-- {
		v := valMsgs[i]
		valRecord, err := fromRESP(v, time.Time{})
		if err != nil {
			log.Printf("%s LPUSH: value iter: %v", ErrCmdPrefix, err)
		}
		newVals[len(valMsgs)-1-i] = valRecord
	}

	record.Value = append(newVals, record.Value.([]*store.Record)...)

	s.store.Set(key, record)
	s.blockingManager.NotifyWatchers(key, record)

	conn.Write([]byte(resp.EncodeInteger(len(record.Value.([]*store.Record)))))
}

// NOTE: Redis seems to default to an empty array when indices are out of bounds
// or when the end index is smaller than the start index. Supporting this for now
// but I feel like it'd be a good idea to let the user know that they've made a
// mistake rather than think they just have an empty array/list in their store.
func (s *Server) handleLrangeCommand(conn net.Conn, msg *resp.Message) {
	if len(msg.Array) != 4 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `LRANGE` command")))
		return
	}

	keyMsg := msg.Array[1]
	startIdxMsg := msg.Array[2]
	endIdxMsg := msg.Array[3]

	key, err := keyMsg.ConvStr()
	if err != nil {
		log.Printf("%s: LRANGE: invalid key name: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Invalid key name type for `LRANGE` command")))
		return
	}

	record, exists := s.store.Get(key)
	if !exists {
		conn.Write([]byte(resp.EncodeArray(0, "")))
		return
	}

	if record.Type != resp.Array {
		log.Printf("%s: LRANGE: invalid type: %s", ErrCmdPrefix, record.Type.String())
		conn.Write([]byte(resp.EncodeSimpleErr("Provided `LRANGE` Key produced non array/list type")))
		return
	}

	startIdx, err := startIdxMsg.ConvInt()
	if err != nil {
		log.Printf("%s: LRANGE: err converting starting index to int: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Invalid start index type for `LRANGE` command")))
		return
	}

	endIdx, err := endIdxMsg.ConvInt()
	if err != nil {
		log.Printf("%s: LRANGE: err converting ending index to int: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Invalid end index type for `LRANGE` command")))
		return
	}

	recArr := record.Value.([]*store.Record)
	startIdx = NormalizeIndex(startIdx, len(recArr))
	endIdx = NormalizeIndex(endIdx, len(recArr))

	if startIdx >= len(recArr) || endIdx < startIdx {
		conn.Write([]byte(resp.EncodeArray(0, "")))
		return
	}

	if endIdx >= len(recArr) {
		endIdx = len(recArr) - 1
	}

	// NOTE: Redis' end index is inclusive, whereas Go's is not, ergo the +1
	endIdx += 1

	toResp, err := toBulkRESPString(recArr[startIdx:endIdx])
	if err != nil {
		log.Printf("%s: LRANGE: to resp string: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Unable to output array")))
		return
	}

	conn.Write([]byte(resp.EncodeArray(len(toResp), toResp...)))
}

func (s *Server) handleRpushCommand(conn net.Conn, msg *resp.Message) {
	if len(msg.Array) <= 2 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `RPUSH` command")))
		return
	}

	keyMsg := msg.Array[1]
	valMsgs := msg.Array[2:]

	key, err := keyMsg.ConvStr()
	if err != nil {
		log.Printf("%s: RPUSH: invalid key name: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Invalid key name type for `RPUSH` command")))
		return
	}

	record, exists := s.store.Get(key)
	if !exists {
		record = &store.Record{
			Type:  resp.Array,
			Value: make([]*store.Record, 0, len(valMsgs)),
		}
	}

	if record.Type != resp.Array {
		log.Printf("%s: RPUSH: invalid type: %s", ErrCmdPrefix, record.Type.String())
		conn.Write([]byte(resp.EncodeSimpleErr("Provided `RPUSH` Key produced non array/list type")))
		return
	}

	for _, v := range valMsgs {
		valRecord, err := fromRESP(v, time.Time{})
		if err != nil {
			log.Printf("%s RPUSH: value iter: %v", ErrCmdPrefix, err)
		}
		record.Value = append(record.Value.([]*store.Record), valRecord)
	}

	s.store.Set(key, record)
	s.blockingManager.NotifyWatchers(key, record)

	conn.Write([]byte(resp.EncodeInteger(len(record.Value.([]*store.Record)))))
}

func (s *Server) handleSetCommand(conn net.Conn, msg *resp.Message) {
	if len(msg.Array) <= 2 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `SET` command")))
		return
	}

	keyMsg := msg.Array[1]
	valMsg := msg.Array[2]

	key, err := keyMsg.ConvStr()
	if err != nil {
		log.Printf("%s: SET: invalid key: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Invalid key type for `SET` command")))
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

	storeRecordValue, err := fromRESP(valMsg, opts.Expiry)
	if err != nil {
		log.Printf("%s SET: store value: %v", ErrCmdPrefix, err)
	}
	s.store.Set(key, storeRecordValue)

	if opts.GET {
		respVal, err := toRESPString(storeRecordValue)
		if err != nil {
			log.Printf("%s: SET: GET value: %v", ErrCmdPrefix, err)
			conn.Write([]byte(resp.EncodeSimpleErr("Unable to retrieve SET value")))
		}
		conn.Write([]byte(respVal))
	} else {
		conn.Write([]byte(resp.EncodeSimpleString("OK")))
	}
	return
}

func (s *Server) handleTypeCommand(conn net.Conn, msg *resp.Message) {
	if len(msg.Array) != 2 {
		conn.Write([]byte(resp.EncodeSimpleErr("Incorrect amount of args for `TYPE` command")))
		return
	}

	keyMsg := msg.Array[1]

	key, err := keyMsg.ConvStr()
	if err != nil {
		log.Printf("%s: TYPE: invalid key: %v", ErrCmdPrefix, err)
		conn.Write([]byte(resp.EncodeSimpleErr("Invalid key type for `TYPE` command")))
		return
	}

	record, _ := s.store.Get(key)
	var stype string

	switch record.Type {
	case resp.Array:
		stype = "list"
	case resp.BulkString, resp.SimpleString:
		stype = "string"
	case resp.Sets:
		stype = "set"
	default:
		stype = "none"
	}
	conn.Write([]byte(resp.EncodeSimpleString(stype)))
}
