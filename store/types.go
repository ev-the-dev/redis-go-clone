package store

import "fmt"

type ErrPrefix string

const (
	ErrGetPrefix ErrPrefix = "store: GET:"
)

type StoreType uint

const (
	Array StoreType = iota
	Booleans
	Errors
	String
	Integer
	Maps
	Nulls
	Sets
	Streams
	None
)

func (t StoreType) String() string {
	switch t {
	case Array:
		return "Array"
	case Booleans:
		return "Booleans"
	case String:
		return "String"
	case Integer:
		return "Integer"
	case Maps:
		return "Maps"
	case Nulls:
		return "Nulls"
	case Sets:
		return "Sets"
	case Streams:
		return "Stream"
	case None:
		return "None"
	default:
		return fmt.Sprintf("UnknownType(%d)", t)
	}
}

type Stream struct {
	root *StreamNode
}

func (s *Stream) Get(id string) (any, bool) {
	node := s.root
	for len(id) > 0 {
		child, _ := node.findChild(id[0])
		if child == nil {
			return nil, false
		}

		shared := child.commonPrefixLen(id)
		if shared != len(child.prefix) {
			return nil, false
		}

		id = id[shared:]
		node = child
	}

	return node.value, node.isLeaf
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
	Fields []string
}

type StreamNode struct {
	prefix   string
	children []*StreamNode
	value    *StreamEntry
	isLeaf   bool
}

func (sn *StreamNode) commonPrefixLen(key string) int {
	l := min(len(sn.prefix), len(key))
	for i := range l {
		if sn.prefix[i] != key[i] {
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
	lo, hi := 0, len(sn.children)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		first := sn.children[mid].prefix[0]
		if first == b {
			return sn.children[mid], mid
		} else if first < b {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}

	// Not found -- lo is where it *would* be inserted
	return nil, lo
}
