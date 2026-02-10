package server

import (
	"net"
	"sync"

	"github.com/ev-the-dev/redis-go-clone/store"
)

type BlockingManager struct {
	queue map[string][]*BlockedClient
	mu    sync.Mutex
}

/* NOTE: We don't need an ID for this struct because we're using it as
* a pointer, ergo we can do checks against the memory address to determine
* when to pop a particular BlockedClient from the above KeyQueue.
 */
type BlockedClient struct {
	conn    net.Conn
	replyCh chan *BlockedClientChanResp
	subs    []string
}

type BlockedClientChanResp struct {
	key string
	rec *store.Record
}

func (bm *BlockingManager) NotifyWatchers(key string, rec *store.Record) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	watchers, exists := bm.queue[key]
	if !exists || len(watchers) == 0 {
		return
	}

	// FIFO
	client := watchers[0]

	select {
	case client.replyCh <- &BlockedClientChanResp{key: key, rec: rec}:
		bm.unregisterClientLocked(client)
	default:
		// Stale client?
		bm.queue[key] = watchers[1:]
	}
}

func (bm *BlockingManager) RegisterClient(bc *BlockedClient) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	for _, key := range bc.subs {
		_, exists := bm.queue[key]
		if !exists {
			bm.queue[key] = make([]*BlockedClient, 0, len(bc.subs))
		}

		bm.queue[key] = append(bm.queue[key], bc)
	}
}

func (bm *BlockingManager) UnregisterClient(bc *BlockedClient) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	bm.unregisterClientLocked(bc)
}

func (bm *BlockingManager) unregisterClientLocked(bc *BlockedClient) {
	for _, keys := range bc.subs {
		clients := bm.queue[keys]

		i := 0
		for _, c := range clients {
			if c != bc {
				clients[i] = c
				i++
			}
		}

		clients = clients[:i]
		bm.queue[keys] = clients
	}
}
