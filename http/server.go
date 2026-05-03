package main

import (
	"fmt"
	"http/internal/request"
	"http/internal/response"
	"log"
	"net"
	"sync/atomic"
)

type Handler func(w *response.Writer, req *request.Request)

type Server struct {
	closed   atomic.Bool
	handler  Handler
	listener net.Listener
}

func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	server := &Server{
		handler:  handler,
		listener: listener,
	}
	go server.listen()
	return server, nil
}

func (s *Server) Close() error {
	err := s.listener.Close()
	s.closed.Store(true)
	return err
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
	responseWriter := response.NewWriter(conn)
	req, err := request.FromReader(conn)
	headers := response.GetDefaultHeaders(0)

	if err != nil {
		responseWriter.WriteStatusLine(response.BadRequest)
		responseWriter.WriteHeaders(headers)
		return
	}

	s.handler(responseWriter, req)
}
