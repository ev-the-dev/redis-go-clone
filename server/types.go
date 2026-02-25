package server

type ErrPrefix string

const (
	ErrAdaptPrefix  ErrPrefix = "server: adapt:"
	ErrBlockPrefix  ErrPrefix = "server: blocking:"
	ErrCmdPrefix    ErrPrefix = "server: cmd:"
	ErrConnPrefix   ErrPrefix = "server: conn:"
	ErrInitPrefix   ErrPrefix = "server: init:"
	ErrStreamPrefix ErrPrefix = "server: stream:"
)

type CmdName string

const (
	BLPOP  CmdName = "BLPOP"
	CONFIG CmdName = "CONFIG"
	ECHO   CmdName = "ECHO"
	GET    CmdName = "GET"
	KEYS   CmdName = "KEYS"
	LLEN   CmdName = "LLEN"
	LPOP   CmdName = "LPOP"
	LPUSH  CmdName = "LPUSH"
	LRANGE CmdName = "LRANGE"
	PING   CmdName = "PING"
	RPUSH  CmdName = "RPUSH"
	SET    CmdName = "SET"
	TYPE   CmdName = "TYPE"
	XADD   CmdName = "XADD"
)

type Stream struct {
	root *StreamNode
}

func (s *Stream) Get(key string) (any, bool) {
	node := s.root
	for len(key) > 0 {
		child, _ := node.findChild(key[0])
		if child == nil {
			return nil, false
		}

		shared := node.commonPrefixLen(child.prefix, key)
		if shared != len(child.prefix) {
			return nil, false
		}

		key = key[shared:]
		node = child
	}

	return node.value, node.isLeaf
}

type StreamNode struct {
	prefix   string
	children []*StreamNode
	value    any
	isLeaf   bool
}

func (sn *StreamNode) commonPrefixLen(childPrefix, key string) int {
	l := min(len(childPrefix), len(key))
	for i := range l {
		if childPrefix[i] != key[i] {
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
