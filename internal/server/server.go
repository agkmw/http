package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"github.com/agkmw/httpfromtcp/internal/request"
	"github.com/agkmw/httpfromtcp/internal/response"
)

type Handler func(w *response.Writer, r *request.Request)

type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

func (e *HandlerError) Write(w *response.Writer) {
	w.WriteStatusLine(e.StatusCode)
	headers := response.GetDefaultHeaders(len(e.Message))
	w.WriteHeaders(headers)
	w.WriteBody([]byte(e.Message))
}

type Server struct {
	listener net.Listener
	handler  Handler
	closed   atomic.Bool
}

func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	server := &Server{
		listener: listener,
		handler:  handler,
	}
	go server.listen()

	return server, nil
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	w := response.NewWriter(conn)
	req, err := request.RequestFromReader(conn)
	if err != nil {
		herr := &HandlerError{response.StatusBadRequest, err.Error()}
		herr.Write(w)
		return
	}
	s.handler(w, req)
}

func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}
