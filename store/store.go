package store

import (
	"sync"
	"time"

	"github.com/ev-the-dev/redis-go-clone/resp"
)

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
		Data: make(map[string]*Record),
	}
}

func (s *Store) Get(k string) (*Record, bool) {
	s.mu.RLock()
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

func (s *Store) Set(k string, v *Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[k] = v
}
