package resp

import (
	"fmt"
	"strings"
)

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
	Map     map[string]*Message // <-- complex keys are serialized to strings
	String  string
}

func (m *Message) ConvStr() (string, error) {
	switch m.Type {
	case Integer:
		return fmt.Sprint(m.Integer), nil
	case BulkString, SimpleString:
		return m.String, nil
	default:
		return "", fmt.Errorf("%s converting %s to string", ErrTypePrefix, m.Type.String())
	}
}

func (m *Message) SerializeKey() (string, error) {
	switch m.Type {
	case Integer:
		return fmt.Sprintf("int:%d", m.Integer), nil
	case Booleans:
		return fmt.Sprintf("bool:%t", m.Boolean), nil
	case BulkString, SimpleString:
		return m.String, nil
	case Nulls:
		return "null", nil
	case Array:
		var parts []string
		for _, elem := range m.Array {
			k, err := elem.SerializeKey()
			if err != nil {
				return "", fmt.Errorf("%s serialize array key of type (%s) to string", ErrTypePrefix, m.Type.String())
			}
			parts = append(parts, k)
		}
		return fmt.Sprintf("arr:[%s]", strings.Join(parts, ",")), nil
	case Maps:
		// NOTE: Maps as keys are theoretically possible in RESP3 but extremely rare.
		// Punting on full implementation for now - can revisit if needed.
		return "", fmt.Errorf("%s maps as keys not currently supported", ErrTypePrefix)
	default:
		return "", fmt.Errorf("%s serialize key of type (%s) to string", ErrTypePrefix, m.Type.String())
	}
}
