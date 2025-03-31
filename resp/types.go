package resp

import "fmt"

type ErrPrefix string

const (
	ErrEncodePrefix   ErrPrefix = "resp: encode:"
	ErrParsePrefix    ErrPrefix = "resp: parse:"
	ErrProtocolPrefix ErrPrefix = "resp: protocol:"
	ErrTypePrefix     ErrPrefix = "resp: type:"
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

func (t RESPType) String() string {
	switch t {
	case Array:
		return "Array"
	case BulkString:
		return "BulkString"
	case Integer:
		return "Integer"
	case SimpleError:
		return "SimpleError"
	case SimpleString:
		return "SimpleString"
	default:
		return fmt.Sprintf("UnknownType(%d)", t)
	}
}

type Message struct {
	Type    RESPType
	Array   []*Message
	Integer int
	Length  int
	String  string
}

func (m *Message) GetString() (string, error) {
	switch m.Type {
	case Integer:
		return fmt.Sprint(m.Integer), nil
	case BulkString, SimpleString:
		return m.String, nil
	default:
		return "", fmt.Errorf("%s converting %s to string", ErrTypePrefix, m.Type.String())
	}
}
