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

func NewStream(id string, fields []*Record) (*Stream, error) {
	id, err := resolveStreamID(id, "")
	if err != nil {
		log.Printf("%s new stream: resolve id: %v", ErrStreamPrefix, err)
		return nil, fmt.Errorf("%s new stream: resolve id: %w", ErrStreamPrefix, err)
	}

	return &Stream{
		lastID: id,
		Root: &StreamNode{
			Prefix: id,
			IsLeaf: true,
			Value: &StreamEntry{
				ID:     id,
				Fields: fields,
			},
		},
	}, nil
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

func (s *Stream) Insert(id string, fields []*Record) error {
	/* TODO:
	*		1. Create StreamEntry from `fields` and `id`
	*		2. append entry as child StreamNode at prefix with StreamEntry containing full ID
	 */

	return nil
}

type streamNodeId struct {
	timestamp *int64
	seq       *int64
}

func parseStreamID(id string) (*streamNodeId, error) {
	if id == "" {
		return &streamNodeId{}, nil
	}

	split := strings.Split(id, "-")
	snId := &streamNodeId{}

	if len(split) == 1 {
		if split[0] == "*" {
			return snId, nil
		}

		timestamp, err := strconv.ParseInt(split[0], 10, 0)
		if err != nil {
			return nil, fmt.Errorf("%s parse stream id: timestamp: %w", ErrStreamPrefix, err)
		}

		snId.timestamp = &timestamp
		return snId, nil
	}

	if len(split) == 2 {
		timestamp, err := strconv.ParseInt(split[0], 10, 0)
		if err != nil {
			return nil, fmt.Errorf("%s parse stream id: timestamp: %w", ErrStreamPrefix, err)
		}
		snId.timestamp = &timestamp

		if split[1] == "*" {
			return snId, nil
		}

		seq, err := strconv.ParseInt(split[1], 10, 0)
		if err != nil {
			return nil, fmt.Errorf("%s parse stream id: sequence: %w", ErrStreamPrefix, err)
		}
		snId.seq = &seq

		return snId, nil
	}

	return nil, fmt.Errorf("%s parse stream id: incorrect id format: %s", ErrStreamPrefix, id)
}

func resolveStreamID(id string, lastId string) (string, error) {
	now := time.Now().UnixMilli()

	snId, err := parseStreamID(id)
	if err != nil {
		return "", err
	}

	lastSnId, err := parseStreamID(lastId)
	if err != nil {
		return "", err
	}

	// This is to cover the case where there isn't a lastID
	// so that the rest of the calculation can still work.
	if lastSnId.timestamp == nil {
		ts := int64(0)
		seq := int64(-1)
		lastSnId.timestamp = &ts
		lastSnId.seq = &seq
	}

	// Full * Scenario
	if snId.timestamp == nil {
		if now > *lastSnId.timestamp {
			return fmt.Sprintf("%d-%d", now, 0), nil
		}

		return fmt.Sprintf("%d-%d", *lastSnId.timestamp, *lastSnId.seq+1), nil
	}

	if snId.timestamp != nil && *snId.timestamp < *lastSnId.timestamp {
		return "", fmt.Errorf("provided ID timestamp for stream is older than current stream ID timestamp")
	}

	// Partial * Scenario
	if snId.seq == nil {
		if *snId.timestamp > *lastSnId.timestamp {
			return fmt.Sprintf("%d-%d", *snId.timestamp, 0), nil
		}

		if *snId.timestamp == *lastSnId.timestamp {
			return fmt.Sprintf("%d-%d", *snId.timestamp, *lastSnId.seq+1), nil
		}
	}

	// Explicit Scenario
	if *snId.timestamp == *lastSnId.timestamp && snId.seq != nil && *snId.seq <= *lastSnId.seq {
		return "", fmt.Errorf("provided ID sequence for stream is older than current stream ID sequence")
	}

	// Redis does not support ID of `0-0`
	if *snId.timestamp == 0 && *snId.seq == 0 {
		return "", fmt.Errorf("invalid ID")
	}

	return fmt.Sprintf("%d-%d", *snId.timestamp, *snId.seq), nil
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
