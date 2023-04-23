package server

import (
	"fmt"
	"net"
	"time"
)

type Server struct {
	listener net.Listener
	quitChan chan struct{}

	activeConn   int
	connCount    int
	connRate     int
	lastConnTime time.Time
}

func (s *Server) GetNumConnCount() int {
	return s.connCount
}

func (s *Server) GetNumActiveConn() int {
	return s.activeConn
}

func (s *Server) GetNumConnRate() int {
	return s.connRate
}

func (s *Server) Start(numListeners int) {
	defer s.listener.Close()

	fmt.Printf("TCP server listening on %s\n", s.listener.Addr().String())

	var doneChan = make(chan struct{}, numListeners)
	for i := 0; i < numListeners; i++ {
		go s.startListener(doneChan)
	}

	for i := 0; i < numListeners; i++ {
		<-doneChan
	}
}

func (s *Server) startListener(doneChan chan<- struct{}) {
	for {
		select {
		case <-s.quitChan:
			doneChan <- struct{}{}
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				continue
			}
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) Stop() {
	close(s.quitChan)
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	s.connCount++
	s.updateConnectionRate()
	s.activeConn++

	var buf [1024]byte
	for {
		err := conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			s.activeConn--
			return
		}
		_, err = conn.Read(buf[:])
		if err != nil {
			s.activeConn--
			return
		}
	}
}

func (s *Server) updateConnectionRate() {
	now := time.Now()
	if now.Sub(s.lastConnTime) >= time.Second {
		s.connRate = s.connCount - 1
		s.lastConnTime = now
	}
}

func NewServer(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return &Server{
		listener:     listener,
		quitChan:     make(chan struct{}),
		lastConnTime: time.Now(),
	}, nil
}
