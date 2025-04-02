package store

import (
	"sync"
	"time"
)

type Store struct {
	data map[string]item
	mu   sync.RWMutex
}

type item struct {
	expiresAt time.Time
	value     string
}

func New() *Store {
	return &Store{
		data: make(map[string]item),
	}
}

func (s *Store) Get(k string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, exists := s.data[k]
	if !exists {
		return "", exists
	}

	if item.expiresAt.IsZero() || time.Now().Before(item.expiresAt) {
		return item.value, exists
	}
	// TODO: if expired, cleanup store?
	return "", false
}

func (s *Store) Set(k, v string, exp time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k] = item{
		expiresAt: exp,
		value:     v,
	}
}
