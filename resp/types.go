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
	Maps
	Nulls
	Sets
	SimpleError
	SimpleString
)

func (t RESPType) String() string {
	switch t {
	case Array:
		return "Array"
	case Booleans:
		return "Booleans"
	case BulkString:
		return "BulkString"
	case Integer:
		return "Integer"
	case Maps:
		return "Maps"
	case Nulls:
		return "Nulls"
	case Sets:
		return "Sets"
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
	Boolean bool
	Integer int
	Length  int
	Map     map[string]*Message
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
