package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"os"
)

func main() {
	addr, err := net.ResolveUDPAddr("udp", "localhost:42069")
	if err != nil {
		slog.Error("Failed to resolve udp addr", "error", err, "addr", addr.String())
		return
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		slog.Error("Failed to prepare a UDP connection", "error", err)
		return
	}
	defer conn.Close()

	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		s, err := r.ReadString('\n')
		if err != nil {
			slog.Error("Failed to read input", "error", err)
			return
		}

		n, err := conn.Write([]byte(s))
		if err != nil {
			slog.Error("Failed to write to the connection", "error", err)
			return
		}
		slog.Info(fmt.Sprintf("Write %d bytes", n))
	}
}
