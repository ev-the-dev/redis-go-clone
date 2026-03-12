package store

import (
	"sync"
	"time"
)

type Store struct {
	data map[string]*Record
	mu   sync.RWMutex
}

type Record struct {
	ExpiresAt time.Time
	Type      StoreType
	Array     []*Record
	Boolean   bool
	Integer   int
	Map       map[string]*Record
	Streams   *Stream
	String    string
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
		return &Record{Type: NilType}, exists
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
