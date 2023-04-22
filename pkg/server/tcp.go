package server

import (
	"net"
	"sync"
	"time"
)

type TCPServer struct {
	numConnPerSec     int32 // number of connection made to server in last second
	numConnCurrentSec int32 //its the number of connection made from last numCOnnPerSec reset
	numActiveConn     int32 // number of active client connected to server
	numTotalConn      int32 // total conn ever made to server
	ipPerSec          int32 // number of unique ip that connected to server in last sec
	ips               map[string]bool
	mutex             sync.Mutex

	listener net.Listener
	stopChan chan struct{}
}

func NewServer(addr string) (*TCPServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &TCPServer{
		listener: listener,
		stopChan: make(chan struct{}),
		ips:      make(map[string]bool),
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

			go s.handleConn(conn)
		}
	}
}

func (s *TCPServer) handleConn(conn net.Conn) {
	buf := make([]byte, 1024)
	defer conn.Close()
	defer func() {
		s.numActiveConn--
	}()

	s.updateIps(conn)

	for {
		conn.SetReadDeadline(<-time.After(7 * time.Second))
		if _, err := conn.Read(buf); err != nil {
			break
		}
	}

}

func (s *TCPServer) updateIps(conn net.Conn) {
	s.mutex.Lock()
	s.ips[conn.RemoteAddr().(*net.TCPAddr).IP.String()] = true
	s.mutex.Unlock()
}

func (s *TCPServer) updateStats() {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-s.stopChan:

			s.mutex.Lock()
			s.ipPerSec = int32(len(s.ips))
			s.ips = make(map[string]bool)
			s.mutex.Unlock()

			s.numConnPerSec = 0
			return
		case <-ticker.C:
			s.mutex.Lock()
			s.ipPerSec = int32(len(s.ips))
			s.ips = make(map[string]bool)
			s.mutex.Unlock()
			s.numConnPerSec = s.numConnCurrentSec
			s.numConnCurrentSec = 0
		}
	}
}
