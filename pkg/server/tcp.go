package server

import (
	"io"
	"kioken/pkg/dynamicpool"
	"net"
	"sync"
	"time"
)

type TcpServer struct {
	addr            string
	numConnPerSec   int
	numActiveConn   int
	numTotalConn    int
	ipPerSecMap     map[string]time.Time
	ipPerSec        int
	ipMutex         sync.Mutex
	listener        net.Listener
	shouldRun       bool
	numConnAcceptor int // number of connection acceptor threads
	pool            dynamicpool.DynamicPool
}

func NewTcpServer(addr string, numConnAcceptor int) *TcpServer {
	return &TcpServer{
		addr:            addr,
		ipPerSecMap:     make(map[string]time.Time),
		numConnAcceptor: numConnAcceptor,
		pool:            *dynamicpool.NewDynamicPool(30000),
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
	s.pool.Start()
	s.shouldRun = true
	var err error
	s.listener, err = net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	for i := 0; i < s.numConnAcceptor; i++ {
		go s.acceptConnections()
	}

	cpsTicker := time.NewTicker(1 * time.Second)

	go func() {
		for s.shouldRun {
			<-cpsTicker.C
			s.ipMutex.Lock()
			now := time.Now()
			for k, v := range s.ipPerSecMap {
				if now.Sub(v) >= time.Second {
					delete(s.ipPerSecMap, k)
				}
			}
			s.ipPerSec = len(s.ipPerSecMap)
			s.numConnPerSec = 0
			s.ipMutex.Unlock()
		}
	}()

	return nil
}

func (s *TcpServer) Stop() error {
	err := s.listener.Close()
	if err != nil {
		return err
	}
	s.pool.Stop()
	s.shouldRun = false
	return nil
}

func (s *TcpServer) acceptConnections() {
	for s.shouldRun {
		conn, err := s.listener.Accept()
		if err != nil {
			continue
		}
		s.pool.Submit(func() {
			s.handleConnection(conn)
		})
	}
}

func (s *TcpServer) handleConnection(conn net.Conn) {

	s.numConnPerSec++
	s.numTotalConn++

	s.ipMutex.Lock()
	if _, ok := s.ipPerSecMap[conn.RemoteAddr().(*net.TCPAddr).IP.String()]; !ok {
		s.ipPerSecMap[conn.RemoteAddr().(*net.TCPAddr).IP.String()] = time.Now()
	}
	s.ipMutex.Unlock()

	defer func() {
		s.numActiveConn--
		if conn != nil {
			conn.Close()
		}
	}()

	s.numActiveConn++

	buf := make([]byte, 1024)
	for s.shouldRun {
		conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		_, err := io.ReadFull(conn, buf)
		if err != nil {
			if err == io.EOF {
				return
			}
			return
		}
	}
}
