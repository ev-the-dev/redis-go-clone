package store

import "sync"

type Store struct {
	data map[string]item
	mu   sync.RWMutex
}

type item struct {
	value string
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

	return item.value, exists
}

func (s *Store) Set(k, v string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k] = item{value: v}
}
