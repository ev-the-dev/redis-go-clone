package rdb

type ErrPrefix string

const (
	ErrLoadPrefix         ErrPrefix = "rdb: load:"
	ErrLengthEncodePrefix ErrPrefix = "rdb: parse: length encoding:"
)
