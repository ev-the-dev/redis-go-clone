package server

import (
	"fmt"
	"net"
	"os"

	"github.com/ev-the-dev/redis-go-clone/config"
	"github.com/ev-the-dev/redis-go-clone/store"
)

type Server struct {
	config *config.Config
	store  *store.Store
}

func New() *Server {
	return &Server{
		config: config.New(),
		store:  store.New(),
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
