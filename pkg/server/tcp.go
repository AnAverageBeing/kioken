package server

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

type Server struct {
	listener     net.Listener
	quitChan     chan struct{}
	activeConn   int32
	connCount    uint64
	connRate     uint64
	lastConnTime time.Time
}

func (s *Server) GetNumConnCount() uint64 {
	return atomic.LoadUint64(&s.connCount)
}

func (s *Server) GetNumActiveConn() int {
	return int(atomic.LoadInt32(&s.activeConn))
}

func (s *Server) GetNumConnRate() uint64 {
	return atomic.LoadUint64(&s.connRate)
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
	defer func() {
		conn.Close()
		atomic.AddInt32(&s.activeConn, -1)
	}()

	atomic.AddUint64(&s.connCount, 1)
	atomic.AddUint64(&s.connRate, 1)
	atomic.AddInt32(&s.activeConn, 1)

	var buf [1024]byte
	for {
		err := conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		if err != nil {
			return
		}
		_, err = conn.Read(buf[:])
		if err != nil {
			return
		}
	}
}

func (s *Server) updateConnectionRate() {
	now := time.Now()
	diff := now.Sub(s.lastConnTime)
	if diff >= time.Second {
		connCount := atomic.LoadUint64(&s.connCount)
		prevCount := connCount - atomic.SwapUint64(&s.connRate, connCount)
		fmt.Printf("Connections made in the last 1 second: %d\n", prevCount)
		s.lastConnTime = now
	}
}

func (s *Server) updateConnectionRateLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.quitChan:
			return
		case <-ticker.C:
			s.updateConnectionRate()
		}
	}
}

func NewServer(addr string) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	s := &Server{
		listener:     listener,
		quitChan:     make(chan struct{}),
		lastConnTime: time.Now(),
	}
	go s.updateConnectionRateLoop()
	return s, nil
}
