package main

import (
	"fmt"
	"log/slog"
	"net"
	"strings"

	"github.com/agkmw/httpfromtcp/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		slog.Error("Failed to listen for TCP connections", "addr", listener.Addr(), "error", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("Failed to accept incoming TCP connection", "error", err)
			return
		}
		slog.Info("Connection accepted", "remote_addr", conn.RemoteAddr())

		req, err := request.RequestFromReader(conn)
		if err != nil {
			slog.Error("Failed to read the request", "error", err)
			return
		}
		printOutput(req)

		slog.Info("Connection closed", "remote_addr", conn.RemoteAddr())
		conn.Close()
	}
}

func printOutput(req *request.Request) {
	var output strings.Builder

	output.WriteString("Request line:\n")
	fmt.Fprintf(&output, "- Method: %s\n", req.RequestLine.Method)
	fmt.Fprintf(&output, "- Target: %s\n", req.RequestLine.RequestTarget)
	fmt.Fprintf(&output, "- Version: %s\n", req.RequestLine.HttpVersion)

	output.WriteString("Headers:\n")
	for k, v := range req.Headers {
		fmt.Fprintf(&output, "- %s: %s\n", k, v)
	}

	output.WriteString("Body:\n")
	output.WriteString(string(req.Body) + "\n")

	fmt.Println(output.String())
}
