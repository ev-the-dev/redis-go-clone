package server

type ErrPrefix string

const (
	ErrCmdPrefix  ErrPrefix = "server: cmd:"
	ErrConnPrefix ErrPrefix = "server: conn:"
)
