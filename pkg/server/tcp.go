package server

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type TCPServer struct {
	numConnPerSec int32
	numActiveConn int32
	numTotalConn  int32
	ipPerSec      int32
	ips           map[string]bool
	mutex         sync.Mutex
	listener      net.Listener
	stopChan      chan bool
}

func NewServer(addr string) (*TCPServer, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &TCPServer{
		listener: listener,
		stopChan: make(chan bool),
		ips:      make(map[string]bool),
	}, nil
}

func (s *TCPServer) Start(numAccept int) {
	wg := &sync.WaitGroup{}

	var cps int32 = 0

	for i := 0; i < numAccept; i++ {
		wg.Add(1)
		go s.acceptConn(wg, &cps)
	}

	go s.updateStats(&cps)

	wg.Wait()
}

func (s *TCPServer) Stop() error {
	s.stopChan <- true
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

func (s *TCPServer) acceptConn(wg *sync.WaitGroup, cps *int32) {
	defer wg.Done()

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

			go s.handleConn(conn, cps)
		}
	}
}

func (s *TCPServer) handleConn(conn net.Conn, cps *int32) {
	defer func() {
		conn.Close()
		atomic.AddInt32(&s.numActiveConn, -1)
	}()

	atomic.AddInt32(cps, 1)

	s.mutex.Lock()
	s.ips[conn.RemoteAddr().(*net.TCPAddr).IP.String()] = true
	s.mutex.Unlock()

	for {
		buf := make([]byte, 1024)
		if _, err := conn.Read(buf); err != nil {
			break
		}
	}
}

func (s *TCPServer) updateStats(cps *int32) {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-s.stopChan:
			s.mutex.Lock()
			s.ipPerSec = int32(len(s.ips))
			s.ips = map[string]bool{}
			s.numConnPerSec = *cps
			*cps = 0
			s.mutex.Unlock()
			return
		case <-ticker.C:
			s.mutex.Lock()
			s.ipPerSec = int32(len(s.ips))
			s.ips = map[string]bool{}
			s.numConnPerSec = atomic.SwapInt32(cps, 0)
			s.mutex.Unlock()
		}
	}
}
