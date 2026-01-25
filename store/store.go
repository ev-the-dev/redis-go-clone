package store

import (
	"log"
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
//
// Additionally the structure needs to change to accomadate
// simple keys that the user can utilize, with maintaining
// the integrity of Expirations.
//
// Perhaps something like:
type StoreData struct {
	Key   *Record
	Value *Record
}

// TODO: Should probably move mutex to the Data instead
// Would need to add mutex to StoreData or Record
type Store struct {
	Data map[string]*StoreData
	mu   sync.RWMutex
}

type Record struct {
	ExpiresAt time.Time
	Type      resp.RESPType
	Value     any
}

func New() *Store {
	return &Store{
		Data: make(map[string]*StoreData),
	}
}

func (s *Store) Get(k *Record) (*Record, bool) {
	s.mu.RLock()
	// NOTE: Might have to change how this data is accessed. Not sure
	// If passing in something like a Map as a key will work to fetch
	// the appropriate record. Perhaps hashes of all the data in the key?
	key, err := s.genKey(k)
	if err != nil {
		log.Printf("%s generate key: %v", ErrGetPrefix, err)
		return nil, false
	}

	item, exists := s.Data[key]
	if !exists {
		s.mu.RUnlock()
		return &Record{}, exists
	}

	isKeyExpired := !item.Key.ExpiresAt.IsZero() && time.Now().After(item.Key.ExpiresAt)
	isValExpired := !item.Value.ExpiresAt.IsZero() && time.Now().After(item.Value.ExpiresAt)
	s.mu.RUnlock()

	if !isValExpired && !isKeyExpired {
		return item.Value, exists
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Checking using write lock in case a write occurred that extended TTL between releasing the Read lock and acquiring this Write lock
	if item, exists := s.Data[key]; exists && time.Now().After(item.Value.ExpiresAt) {
		delete(s.Data, key)
	}

	return &Record{}, false
}

func (s *Store) Set(k *Record, v *Record) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[k] = v
}

func (s *Store) genKey(r *Record) (string, error) {

}
