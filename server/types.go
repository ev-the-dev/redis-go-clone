package server

type ErrPrefix string

const (
	ErrCmdPrefix   ErrPrefix = "server: cmd:"
	ErrConnPrefix  ErrPrefix = "server: conn:"
	ErrInitPrefix  ErrPrefix = "server: init:"
	ErrAdaptPrefix ErrPrefix = "server: adapt:"
)
