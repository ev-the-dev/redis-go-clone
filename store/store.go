package store

import (
	"sync"
	"time"
)

type Store struct {
	Data map[string]record
	mu   sync.RWMutex
}

type record struct {
	ExpiresAt time.Time
	Value     string
}

func New() *Store {
	return &Store{
		Data: make(map[string]record),
	}
}

func (s *Store) Get(k string) (record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, exists := s.Data[k]
	if !exists {
		return record{}, exists
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
	return record{}, false
}

func (s *Store) Set(k, v string, exp time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[k] = record{
		ExpiresAt: exp,
		Value:     v,
	}
}
