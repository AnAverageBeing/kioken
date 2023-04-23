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
	totalInBytes uint64    // total inbound bytes received
	lastCalcTime time.Time // last time inbound data rate was calculated
	inDataRate   float64   // inbound data rate in MB/s
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

func (s *Server) GetInDataRate() float64 {
	return s.inDataRate
}

func (s *Server) Start(numListeners int) {
	var i int
	for i = 0; i < numListeners; i++ {
		go s.startListener()
	}

	go s.updateConnPerSec()
	go s.updateInDataRate()
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
			s.pool.SubmitTask(func() { s.handleConnection(conn) }, 1*time.Second)
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
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		atomic.AddUint64(&s.totalInBytes, uint64(n))
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

func (s *Server) updateInDataRate() {
	for {
		select {
		case <-s.quitChan:
			return
		case <-time.After(1 * time.Second):
			totalInBytes := atomic.LoadUint64(&s.totalInBytes)
			duration := time.Since(s.lastCalcTime).Seconds()
			inDataRate := float64(totalInBytes) / (duration * 1024 * 1024)
			s.inDataRate = inDataRate
			atomic.StoreUint64(&s.totalInBytes, 0)
			s.lastCalcTime = time.Now()
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
