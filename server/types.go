package server

import "fmt"

type ErrPrefix string

const (
	ErrAdaptPrefix  ErrPrefix = "server: adapt:"
	ErrBlockPrefix  ErrPrefix = "server: blocking:"
	ErrCmdPrefix    ErrPrefix = "server: cmd:"
	ErrConnPrefix   ErrPrefix = "server: conn:"
	ErrInitPrefix   ErrPrefix = "server: init:"
	ErrStreamPrefix ErrPrefix = "server: stream:"
)

type CmdName string

const (
	BLPOP  CmdName = "BLPOP"
	CONFIG CmdName = "CONFIG"
	ECHO   CmdName = "ECHO"
	GET    CmdName = "GET"
	KEYS   CmdName = "KEYS"
	LLEN   CmdName = "LLEN"
	LPOP   CmdName = "LPOP"
	LPUSH  CmdName = "LPUSH"
	LRANGE CmdName = "LRANGE"
	PING   CmdName = "PING"
	RPUSH  CmdName = "RPUSH"
	SET    CmdName = "SET"
	TYPE   CmdName = "TYPE"
	XADD   CmdName = "XADD"
)

type StreamNode struct {
	prefix   string
	children []*StreamNode
	value    any
	isLeaf   bool
}

func (sn *StreamNode) Get(id string) (*StreamNode, error) {
	if sn.prefix == id {
		return sn, nil
	}

	for c := range sn.children {

	}

	return nil, fmt.Errorf("%s unable to find node at (%s)", ErrStreamPrefix, id)
}
