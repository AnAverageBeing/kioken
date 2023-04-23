package server

import (
	"kioken/pkg/pool"
	"net"
	"sync/atomic"
	"time"
)

type Server struct {
	listener net.Listener
	quitChan chan struct{}
	pool     pool.Pool

	activeConn   int32  //total connection that are active
	connCount    uint64 // total connection made to server
	ConnPerSec   uint64 // number of client connected in last 1 sec (updated at end of each sec)
	localCPS     uint64 // number of client connected in last 1 sec (updated every sec)
	lastConnTime time.Time
}

func (s *Server) GetNumConnCount() uint64 {
	return atomic.LoadUint64(&s.connCount)
}

func (s *Server) GetNumActiveConn() int {
	return int(atomic.LoadInt32(&s.activeConn))
}

func (s *Server) GetNumConnRate() uint64 {
	return atomic.LoadUint64(&s.ConnPerSec)
}

func (s *Server) Start(numListeners int) {
	var i int
	for i = 0; i < numListeners; i++ {
		go s.startListener()
	}

	go s.updateConnPerSec()
}

func (s *Server) startListener() {
	for {
		conn, err := s.listener.Accept()

		if err != nil {
			select {
			case <-s.quitChan:
				return
			default:
				continue
			}
		}

		select {
		case <-s.quitChan:
			return
		default:
			s.pool.SubmitTask(func() { s.handleConnection(conn) }, 5*time.Second)
		}

	}
}

func (s *Server) Stop() {
	close(s.quitChan)
	s.listener.Close()
	s.pool.Shutdown()
}

func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		conn.Close()
		atomic.AddInt32(&s.activeConn, -1)
	}()

	atomic.AddUint64(&s.connCount, 1)
	atomic.AddUint64(&s.localCPS, 1)
	atomic.AddInt32(&s.activeConn, 1)

	buf := make([]byte, 1024)
	for {
		err := conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		if err != nil {
			return
		}
		_, err = conn.Read(buf)
		if err != nil {
			return
		}
	}
}

func (s *Server) updateConnPerSec() {
	for {
		select {
		case <-s.quitChan:
			return
		case <-time.After(time.Second):
			s.ConnPerSec = atomic.LoadUint64(&s.localCPS)
			atomic.StoreUint64(&s.localCPS, 0)
		}
	}
}

func NewServer(addr string, poolSize int) (*Server, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	return &Server{
		listener:     listener,
		quitChan:     make(chan struct{}),
		lastConnTime: time.Now(),
		pool:         *pool.New(poolSize),
	}, nil
}
