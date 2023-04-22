package server

import (
	"net"
	"sync"
	"time"

	"github.com/panjf2000/ants/v2"
)

type TCPServer struct {
	numConnPerSec     int32 // number of connection made to server in last second
	numConnCurrentSec int32 //its the number of connection made from last numCOnnPerSec reset
	numActiveConn     int32 // number of active client connected to server
	numTotalConn      int32 // total conn ever made to server
	ipPerSec          int32 // number of unique ip that connected to server in last sec
	ips               map[string]bool

	listener net.Listener
	stopChan chan struct{}
	pool     ants.Pool
}

func NewServer(addr string, numWorkers int) (*TCPServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	pool, err := ants.NewPool(numWorkers, ants.WithNonblocking(true), ants.WithPreAlloc(true))
	if err != nil {
		return nil, err
	}

	return &TCPServer{
		listener: listener,
		stopChan: make(chan struct{}),
		ips:      make(map[string]bool),
		pool:     *pool,
	}, nil
}

func (s *TCPServer) Start(numAccept int) {
	var wg sync.WaitGroup
	wg.Add(numAccept)

	for i := 0; i < numAccept; i++ {
		go func() {
			defer wg.Done()
			s.acceptConn()
		}()
	}

	go s.updateStats()

	wg.Wait()
}

func (s *TCPServer) Stop() error {
	close(s.stopChan)
	return s.listener.Close()
}

func (s *TCPServer) GetNumConnPerSec() int {
	return int(s.numConnPerSec)
}

func (s *TCPServer) GetNumActiveConn() int {
	return int(s.numActiveConn)
}

func (s *TCPServer) GetNumTotalConn() int {
	return int(s.numTotalConn)
}

func (s *TCPServer) GetIpPerSec() int {
	return int(s.ipPerSec)
}

func (s *TCPServer) acceptConn() {
	for {
		select {
		case <-s.stopChan:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				continue
			}

			s.numTotalConn++
			s.numActiveConn++
			s.numConnCurrentSec++

			s.pool.Submit(func() { s.handleConn(conn) })
		}
	}
}

func (s *TCPServer) handleConn(conn net.Conn) {
	buf := make([]byte, 1024)
	defer conn.Close()

	s.updateIps(conn)

	for {
		conn.SetReadDeadline(<-time.After(7 * time.Second))
		if _, err := conn.Read(buf); err != nil {
			break
		}
	}

	s.numActiveConn--
}

func (s *TCPServer) updateIps(conn net.Conn) {
	s.ips[conn.RemoteAddr().(*net.TCPAddr).IP.String()] = true
}

func (s *TCPServer) updateStats() {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-s.stopChan:
			s.ipPerSec = int32(len(s.ips))
			s.ips = make(map[string]bool)
			s.numConnPerSec = 0
			return
		case <-ticker.C:
			s.ipPerSec = int32(len(s.ips))
			s.ips = make(map[string]bool)
			s.numConnPerSec = s.numConnCurrentSec
			s.numConnCurrentSec = 0
		}
	}
}
