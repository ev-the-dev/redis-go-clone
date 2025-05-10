package server

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

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

	memStore := store.New()

	err := rdb.Load(filepath.Join(cfg.Dir, cfg.DBFilename), memStore)
	if err != nil {
		fmt.Printf("%s new: %v\n", ErrInitPrefix, err)
	}

	return &Server{
		config: cfg,
		store:  memStore,
	}
}

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
			fmt.Printf("%s client: %v\n", ErrConnPrefix, err.Error())
			continue
		}

		go s.handleConnection(conn)
	}
}
