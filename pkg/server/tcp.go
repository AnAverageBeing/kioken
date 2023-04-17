package server

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type TCPServer struct {
	numConnPerSec int32
	localCPS      int32
	numActiveConn int32
	numTotalConn  int32
	ipPerSec      int32
	ips           map[string]bool
	mutex         sync.Mutex
	listener      net.Listener
	stopChan      chan struct{}
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
	return int(atomic.LoadInt32(&s.numConnPerSec))
}

func (s *TCPServer) GetNumActiveConn() int {
	return int(atomic.LoadInt32(&s.numActiveConn))
}

func (s *TCPServer) GetNumTotalConn() int {
	return int(atomic.LoadInt32(&s.numTotalConn))
}

func (s *TCPServer) GetIpPerSec() int {
	return int(atomic.LoadInt32(&s.ipPerSec))
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

			atomic.AddInt32(&s.numTotalConn, 1)
			atomic.AddInt32(&s.numActiveConn, 1)
			atomic.AddInt32(&s.localCPS, 1)

			go func() {
				s.handleConn(conn)
				if s.GetNumActiveConn() > 0 {
					atomic.AddInt32(&s.numActiveConn, -1)
				}
			}()
		}
	}
}

func (s *TCPServer) handleConn(conn net.Conn) {
	buf := make([]byte, 1024)
	defer conn.Close()

	for {
		if _, err := conn.Read(buf); err != nil {
			break
		}
	}

	s.updateIps(conn)
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
			s.numConnPerSec = 0
			s.mutex.Unlock()
			return
		case <-ticker.C:
			s.mutex.Lock()
			s.ipPerSec = int32(len(s.ips))
			s.ips = make(map[string]bool)
			atomic.SwapInt32(&s.numConnPerSec, s.localCPS)
			atomic.StoreInt32(&s.localCPS, 0)
			s.mutex.Unlock()
		}
	}
}
