package store

import (
	"sync"
	"time"

	"github.com/ev-the-dev/redis-go-clone/resp"
)

// TODO:
// NOTE:
// Store.Data needs to have the key be of type 'string'.
// The *Record embeds an expiry that would make it impossible
// to GET any data in the store without knowing all the data
// that the key is comprised off. Ergo, we need to make the key
// the hash of JUST the *Record.Value (need to loop/recurse) to
// ensure we're appending nested struct's *Record.Value as well
type Store struct {
	Data map[string]*Record
	mu   sync.RWMutex
}

type Record struct {
	ExpiresAt time.Time
	Type      resp.RESPType
	Value     any
}

func New() *Store {
	return &Store{
		Data: make(map[*Record]*Record),
	}
}

func (s *Store) Get(k *Record) (*Record, bool) {
	s.mu.RLock()
	// NOTE: Might have to change how this data is accessed. Not sure
	// If passing in something like a Map as a key will work to fetch
	// the appropriate record. Perhaps hashes of all the data in the key?
	item, exists := s.Data[k]
	if !exists {
		s.mu.RUnlock()
		return &Record{}, exists
	}

	isExpired := !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt)
	s.mu.RUnlock()

	if !isExpired {
		return item, exists
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Checking using write lock in case a write occurred that extended TTL between releasing the Read lock and acquiring this Write lock
	if item, exists := s.Data[k]; exists && time.Now().After(item.ExpiresAt) {
		delete(s.Data, k)
	}

	return &Record{}, false
}

func (s *Store) Set(k *Record, v *Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[k] = v
}
