package server

type ErrPrefix string

const (
	ErrCmdPrefix   ErrPrefix = "server: cmd:"
	ErrConnPrefix  ErrPrefix = "server: conn:"
	ErrInitPrefix  ErrPrefix = "server: init:"
	ErrAdaptPrefix ErrPrefix = "server: adapt:"
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
)
