package server

import (
	"net"

	"github.com/ev-the-dev/redis-go-clone/store"
)

type BlockingManager struct {
	queue map[string][]*BlockedClient
}

/* NOTE: We don't need an ID for this struct because we're using it as
* a pointer, ergo we can do checks against the memory address to determine
* when to pop a particular BlockedClient from the above KeyQueue.
 */
type BlockedClient struct {
	conn    net.Conn
	replyCh chan *store.Record
	subs    []string
}

func (bm *BlockingManager) RegisterClient(bc *BlockedClient) {
	for _, key := range bc.subs {
		_, exists := bm.queue[key]
		if !exists {
			bm.queue[key] = make([]*BlockedClient, 0, len(bc.subs))
		}

		bm.queue[key] = append(bm.queue[key], bc)
	}
}

func (bm *BlockingManager) UnregisterClient(bc *BlockedClient) {
	for _, keys := range bc.subs {
		clients := bm.queue[keys]

		i := 0
		for _, c := range clients {
			if clients[i] != bc {
				clients[i] = c
				i++
			}
		}

		clients = clients[:i]
	}
}
