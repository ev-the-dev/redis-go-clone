package server

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/ev-the-dev/redis-go-clone/config"
	"github.com/ev-the-dev/redis-go-clone/rdb"
	"github.com/ev-the-dev/redis-go-clone/store"
)

type Server struct {
	config *config.Config
	store  *store.Store
}

func New(cfg *config.Config) *Server {
	if cfg == nil {
		cfg = config.New()
	}

	memStore, err := initStore(cfg)
	if err != nil {
		log.Fatal(err)
	}

	return &Server{
		config: cfg,
		store:  memStore,
	}
}

// TODO: need to rethink how I handle connections. I need to
// pool them together to be able to handle concurrent conns
// trying to access the same data with blocks. This might also
// improve the signature of each handler as they could probably just
// return a RESP string and then some central orchestrator uses that
// output to write to the connection(s).
//
// The only situation I could see this causing issues is we'd need to
// write to the connection multiple times for a command, then using
// the returns instead of passing the connection directly could pose
// an issue in this regard.
func (s *Server) Start() {
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Printf("%s port: %v\n", ErrConnPrefix, err)
		os.Exit(1)
	}

	fmt.Println("Listening on port: 6379")

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("%s client: %v\n", ErrConnPrefix, err.Error())
			continue
		}

		// TODO: Add a cancellation context
		go s.handleConnection(conn)
	}
}

func initStore(cfg *config.Config) (*store.Store, error) {
	store := store.New()

	entriesCh := make(chan *rdb.Entry, 10)
	go func() {
		err := rdb.Load(filepath.Join(cfg.Dir, cfg.DBFilename), entriesCh)
		if err != nil {
			log.Printf("%s store init: %v\n", ErrInitPrefix, err)
		}
	}()

	for {
		select {
		case entry, ok := <-entriesCh:
			if !ok {
				return store, nil
			}
			storeRecord, err := fromRDB(entry)
			if err != nil {
				log.Printf("%s: store init: fromRDB: %v", ErrInitPrefix, err)
			}
			store.Set(entry.Key, storeRecord)
		case <-time.After(3 * time.Second):
			return nil, fmt.Errorf("%s store init: timeout", ErrInitPrefix)
		}
	}
}
