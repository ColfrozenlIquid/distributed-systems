package vshttp

import (
	"log"
	"net"
	"sync/atomic"
	"vshttp/internal/request"
)

type Handler interface {
	ServeHTTP(w ResponseWriter, r *Request)
}

type Server struct {
	addr     string
	closed   atomic.Bool
	handler  Handler
	listener net.Listener
}

func NewServer(addr string, handler Handler) *Server {
	return &Server{
		addr:    addr,
		handler: handler,
	}
}

func Serve(addr string, handler Handler) (*Server, error) {
	server := NewServer(addr, handler)
	if err := server.Start(); err != nil {
		return nil, err
	}
	return server, nil
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = listener
	go s.listen()
	return nil
}

func (s *Server) Close() error {
	s.closed.Store(true)
	if s.listener == nil {
		return nil
	}
	return s.listener.Close()
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

	w := newResponseWriter(conn)
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("Panic while handling request: %v", recovered)
			w.writeError(InternalServerError, "internal server error")
		}
		if err := w.finish(); err != nil {
			log.Printf("Error writing response: %v", err)
		}
	}()

	rawReq, err := request.FromReader(conn)
	if err != nil {
		w.writeError(BadRequest, "bad request")
		return
	}

	if s.handler == nil {
		w.writeError(NotFound, "not found")
		return
	}

	s.handler.ServeHTTP(w, newRequest(rawReq))
}
