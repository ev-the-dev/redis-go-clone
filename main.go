package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/ev-the-dev/redis-go-clone/resp"
)

func main() {

	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	fmt.Println("Listening on port: 6379")

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection::: ", err.Error())
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("Error reading client input::: %v\n", err)
			return
		}

		line = strings.TrimSpace(line)
		fmt.Printf("LINE: %s\n", line)
		cmd, arg := parseCmdArgs(line)

		fmt.Printf("CMD: %s\n", cmd)
		fmt.Printf("ARG: %s\n", arg)

		switch cmd {
		case "PING":
			conn.Write([]byte("+PONG\r\n"))
		case "ECHO":
			if len(arg) < 1 {
				conn.Write([]byte(resp.EncodeSimpleErr("missing argument for ECHO")))
			} else {
				conn.Write([]byte(resp.EncodeBulkString(arg)))
			}
		default:
			conn.Write([]byte(resp.EncodeSimpleErr("unknown command")))
		}
	}
}

func parseCmdArgs(line string) (string, string) {
	parts := strings.Split(line, " ")
	if len(parts) == 0 {
		return "", ""
	}

	return strings.ToUpper(parts[0]), strings.Join(parts[1:], " ")
}
