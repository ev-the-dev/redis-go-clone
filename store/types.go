package store

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

type ErrPrefix string

const (
	ErrStreamPrefix ErrPrefix = "store: stream:"
)

type StoreType uint

const (
	ArrayType StoreType = iota
	BooleanType
	ErrorType
	StringType
	IntegerType
	MapType
	NilType
	SetType
	StreamType
	NoneType
)

func (t StoreType) String() string {
	switch t {
	case ArrayType:
		return "Array"
	case BooleanType:
		return "Booleans"
	case StringType:
		return "String"
	case IntegerType:
		return "Integer"
	case MapType:
		return "Maps"
	case NilType:
		return "Nulls"
	case SetType:
		return "Sets"
	case StreamType:
		return "Stream"
	case NoneType:
		return "None"
	default:
		return fmt.Sprintf("UnknownType(%d)", t)
	}
}

type Stream struct {
	Root   *StreamNode
	lastID string
}

func NewStream(id string, fields []*Record) *Stream {
	id, err := resolveStreamID(id)
	if err != nil {
		log.Printf("%s new stream: normalize id: %v", ErrStreamPrefix, err)
	}

	return &Stream{
		Root: &StreamNode{
			Prefix: id,
			IsLeaf: true,
			Value: &StreamEntry{
				ID:     id,
				Fields: fields,
			},
		},
	}
}

// NOTE: sequences will most likely never get very large.
// Can consider using a smaller data type instead of int64,
// perhaps uint16, or something along those lines.
func parseStreamId(id string) (int64, int64, error) {
	split := strings.Split(id, "-")
	ts, err := strconv.ParseInt(split[0], 10, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("%s parse stream id: timestamp: %w", ErrStreamPrefix, err)
	}

	seq, err := strconv.ParseInt(split[1], 10, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("%s parse stream id: sequence: %w", ErrStreamPrefix, err)
	}

	return ts, seq, nil
}

func resolveStreamID(id string, lastId string) (string, error) {
	now := time.Now().UnixMilli()

	lastTS, lastSeq, err := parseStreamId(lastId)
	if err != nil {
		return "", err
	}

	// Full * Scenario
	if id == "*" {
		if now > lastTS {
			return fmt.Sprintf("%d-%d%d", now, 0, 0), nil
		}

		if lastSeq >= 9 {
			return fmt.Sprintf("%d-%d", lastTS, lastSeq+1), nil
		} else {
			return fmt.Sprintf("%d-%d%d", lastTS, 0, lastSeq+1), nil
		}
	}

	ts, seq, err := parseStreamId(id)
	if err != nil {
		return "", err
	}
	if ts < lastTS {
		return "", fmt.Errorf("provided ID for stream is older than current stream ID")
	}

	// Partial * Scenario
	if seq == "*" {

	}
}

func (s *Stream) Get(id string) (any, bool) {
	node := s.Root
	for len(id) > 0 {
		child, _ := node.findChild(id[0])
		if child == nil {
			return nil, false
		}

		shared := child.commonPrefixLen(id)
		if shared != len(child.Prefix) {
			return nil, false
		}

		id = id[shared:]
		node = child
	}

	return node.Value, node.IsLeaf
}

func (s *Stream) Insert(id string, fields []string) error {
	/* TODO:
	*		1. Create StreamEntry from `fields` and `id`
	*		2. append entry as child StreamNode at prefix with StreamEntry containing full ID
	 */

	return nil
}

type StreamEntry struct {
	ID     string
	Fields []*Record
}

type StreamNode struct {
	Prefix   string
	Children []*StreamNode
	Value    *StreamEntry
	IsLeaf   bool
}

func (sn *StreamNode) commonPrefixLen(key string) int {
	l := min(len(sn.Prefix), len(key))
	for i := range l {
		if sn.Prefix[i] != key[i] {
			return i
		}
	}

	return l
}

/*
* The Radix tree that this traverses should always parse out common
* prefixes, so each child should never have the same first byte, hence
* why the comparison between `first` and `b`
 */
func (sn *StreamNode) findChild(b byte) (*StreamNode, int) {
	lo, hi := 0, len(sn.Children)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		first := sn.Children[mid].Prefix[0]
		if first == b {
			return sn.Children[mid], mid
		} else if first < b {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}

	// Not found -- lo is where it *would* be inserted
	return nil, lo
}
