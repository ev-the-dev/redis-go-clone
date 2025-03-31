package main

import (
	"github.com/ev-the-dev/redis-go-clone/server"
)

func main() {
	s := server.New()
	s.Start()
}
