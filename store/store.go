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
	defer s.mu.RUnlock()
	item, exists := s.Data[k]
	if !exists {
		return &Record{}, exists
	}

	if item.ExpiresAt.IsZero() || time.Now().Before(item.ExpiresAt) {
		return item, exists
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Checking using write lock in case a write occurred that extended TTL
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
