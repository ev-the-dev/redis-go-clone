package store

import (
	"sync"
	"time"

	"github.com/ev-the-dev/redis-go-clone/resp"
)

type Store struct {
	data map[string]*Record
	mu   sync.RWMutex
}

// NOTE: Consider refactoring this struct to look more like
// `resp.Message` to avoid having to perform a lot of runtime
// type checks.
type Record struct {
	ExpiresAt time.Time
	Type      resp.RESPType
	Value     any
}

func New() *Store {
	return &Store{
		data: make(map[string]*Record),
	}
}

func (s *Store) Get(k string) (*Record, bool) {
	s.mu.RLock()
	item, exists := s.data[k]
	if !exists {
		s.mu.RUnlock()
		return &Record{Type: resp.None}, exists
	}

	isExpired := !item.ExpiresAt.IsZero() && time.Now().After(item.ExpiresAt)
	s.mu.RUnlock()

	if !isExpired {
		return item, exists
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Checking using write lock in case a write occurred that extended TTL between releasing the Read lock and acquiring this Write lock
	if item, exists := s.data[k]; exists && time.Now().After(item.ExpiresAt) {
		delete(s.data, k)
	}

	return &Record{}, false
}

func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, len(s.data))
	i := 0
	for k := range s.data {
		keys[i] = k
		i++
	}

	return keys
}

func (s *Store) Set(k string, v *Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k] = v
}
