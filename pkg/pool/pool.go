package pool

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Task is a function that can be executed by the goroutines in the pool.
type Task func()

var ErrPoolShutdown = errors.New("pool is shutdown")

// Pool is a dynamically growing goroutine pool.
type Pool struct {
	tasks        chan Task
	workers      []*worker
	capacity     int
	mutex        sync.Mutex
	waitGroup    sync.WaitGroup
	workerTimers map[*worker]*time.Timer
	shutdown     bool
}

// worker is a goroutine that waits for tasks to execute.
type worker struct {
	pool     *Pool
	lastUsed time.Time
}

// New creates a new goroutine pool with the specified initial capacity.
func New(capacity int) *Pool {
	pool := &Pool{
		tasks:        make(chan Task, capacity),
		workers:      make([]*worker, 0, capacity),
		capacity:     capacity,
		workerTimers: make(map[*worker]*time.Timer),
	}
	for i := 0; i < capacity; i++ {
		w := &worker{pool: pool}
		pool.workers = append(pool.workers, w)
		pool.workerTimers[w] = time.AfterFunc(1*time.Minute, w.stopIfIdle)
		go w.run()
	}
	return pool
}

// SubmitTask adds a new task to the pool with an optional timeout.
func (p *Pool) SubmitTask(task Task, timeout time.Duration) error {
	if p.shutdown {
		return ErrPoolShutdown
	}

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	}

	p.waitGroup.Add(1)
	select {
	case p.tasks <- func() {
		defer p.waitGroup.Done()
		task()
	}:
		for _, w := range p.workers {
			if p.workerTimers[w].Reset(0) {
				break
			}
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SetCapacity sets the maximum number of workers that the pool can have.
func (p *Pool) SetCapacity(capacity int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if capacity < p.capacity {
		diff := p.capacity - capacity
		for i := 0; i < diff; i++ {
			w := p.workers[len(p.workers)-1]
			delete(p.workerTimers, w)
			w.stop()
			p.workers = p.workers[:len(p.workers)-1]
		}
	} else if capacity > p.capacity {
		diff := capacity - p.capacity
		for i := 0; i < diff; i++ {
			w := &worker{pool: p}
			p.workerTimers[w] = time.AfterFunc(1*time.Minute, w.stopIfIdle)
			p.workers = append(p.workers, w)
			go w.run()
		}
	}
	p.capacity = capacity
}

// Wait waits for all tasks to finish executing.
func (p *Pool) Wait() {
	p.waitGroup.Wait()
}

// Shutdown stops all workers and waits for all tasks to finish executing.
func (p *Pool) Shutdown() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.shutdown = true

	for _, w := range p.workers {
		delete(p.workerTimers, w)
		w.stop()
	}

	close(p.tasks)
	p.Wait()
}

// worker.run waits for tasks to execute.
func (w *worker) run() {
	for task := range w.pool.tasks {
		w.lastUsed = time.Now()
		task()
		if w.pool.shutdown {
			w.stop()
			return
		}
		w.pool.workerTimers[w].Reset(1 * time.Minute)
	}
}

// worker.stop stops the worker from waiting for tasks to execute.
func (w *worker) stop() {
	close(w.pool.tasks)
}

// worker.stopIfIdle stops the worker if it has been idle for more than one minute.
func (w *worker) stopIfIdle() {
	w.pool.mutex.Lock()
	defer w.pool.mutex.Unlock()

	if len(w.pool.tasks) == 0 && time.Since(w.lastUsed) > 1*time.Minute {
		delete(w.pool.workerTimers, w)
		w.stop()
		for i, worker := range w.pool.workers {
			if worker == w {
				w.pool.workers = append(w.pool.workers[:i], w.pool.workers[i+1:]...)
				break
			}
		}
		w.pool.capacity--
	} else {
		w.pool.workerTimers[w].Reset(1 * time.Minute)
	}
}
