package server

import (
	"io"
	"net"
	"sync"
	"time"
)

type TcpServer struct {
	addr          string
	numConnPerSec int
	numActiveConn int
	numTotalConn  int
	ipPerSecMap   map[string]bool
	ipPerSec      int
	ipMutex       sync.Mutex
	listener      net.Listener
	shouldRun     bool
}

func NewTcpServer(addr string) *TcpServer {
	return &TcpServer{
		addr:        addr,
		ipPerSecMap: make(map[string]bool),
	}
}

func (s *TcpServer) GetNumConnPerSec() int {
	return s.numConnPerSec
}

func (s *TcpServer) GetNumActiveConn() int {
	return s.numActiveConn
}

func (s *TcpServer) GetNumTotalConn() int {
	return s.numTotalConn
}

func (s *TcpServer) GetIpPerSec() int {
	return s.ipPerSec
}

func (s *TcpServer) Start() error {
	s.shouldRun = true
	var err error
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	cpsTicker := time.NewTicker(1 * time.Second)
	connCh := make(chan net.Conn)

	go s.acceptConnections(connCh)

	go func() {
		for s.shouldRun {
			select {
			case <-cpsTicker.C:
				s.ipMutex.Lock()
				s.ipPerSec = len(s.ipPerSecMap)
				s.numConnPerSec = 0
				s.ipPerSecMap = make(map[string]bool)
				s.ipMutex.Unlock()
			case conn := <-connCh:
				go s.handleConnection(conn)
				s.numTotalConn++
				s.ipMutex.Lock()
				s.ipPerSecMap[conn.RemoteAddr().String()] = true
				s.ipMutex.Unlock()
				s.numConnPerSec++
			}
		}
	}()

	return nil
}

func (s *TcpServer) Stop() error {
	err := s.listener.Close()
	if err != nil {
		return err
	}
	s.shouldRun = false
	return nil
}

func (s *TcpServer) acceptConnections(connCh chan net.Conn) {
	for s.shouldRun {
		conn, err := s.listener.Accept()
		if err != nil {
			continue
		}
		connCh <- conn
	}
}

func (s *TcpServer) handleConnection(conn net.Conn) {
	s.numActiveConn++

	s.ipMutex.Lock()
	s.ipPerSecMap[conn.RemoteAddr().String()] = true
	s.ipMutex.Unlock()

	defer func() {
		conn.Close()
		s.numActiveConn--
	}()

	buf := make([]byte, 1024)
	for s.shouldRun {
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		_, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				return
			}
			return
		}
	}
}
