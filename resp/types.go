package resp

type ErrPrefix string

const (
	ErrEncodePrefix   ErrPrefix = "resp: encode:"
	ErrParsePrefix    ErrPrefix = "resp: parse:"
	ErrProtocolPrefix ErrPrefix = "resp: protocol:"
)

type RESPType uint

const (
	Array RESPType = iota
	Booleans
	BulkErrors
	BulkString
	Integer
	Nulls
	SimpleError
	SimpleString
)

type Value struct {
	Type    RESPType
	Array   []*Value
	Integer int
	Length  int
	String  string
}
