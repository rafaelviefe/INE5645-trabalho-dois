package tcp

import (
	"net"
	"time"
)

type HandlerFunc func(request []byte) []byte

type Server struct {
	address    string
	maxWorkers int
	handler    HandlerFunc
}

func NewServer(address string, maxWorkers int, handler HandlerFunc) *Server {
	return &Server{
		address:    address,
		maxWorkers: maxWorkers,
		handler:    handler,
	}
}

func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	defer listener.Close()

	semaphore := make(chan struct{}, s.maxWorkers)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		semaphore <- struct{}{}

		go func(c net.Conn) {
			defer func() {
				<-semaphore
				c.Close()
			}()

			c.SetReadDeadline(time.Now().Add(10 * time.Second))
			req, err := ReadMessage(c)
			if err != nil {
				return
			}

			res := s.handler(req)

			if res != nil {
				c.SetWriteDeadline(time.Now().Add(10 * time.Second))
				_ = WriteMessage(c, res)
			}
		}(conn)
	}
}
