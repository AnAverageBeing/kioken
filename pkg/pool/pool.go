package pool

import (
	"runtime"
	"sync"
	"time"
)

// Task is a function that can be executed by the goroutines in the pool.
type Task func()

// Pool is a dynamically growing goroutine pool.
type Pool struct {
	tasks    chan Task
	workers  []*worker
	mutex    sync.Mutex
	shutdown bool
}

// worker is a goroutine that waits for tasks to execute.
type worker struct {
	pool *Pool
}

// New creates a new goroutine pool with the specified initial capacity.
func New(capacity int) *Pool {
	pool := &Pool{
		tasks:   make(chan Task),
		workers: make([]*worker, capacity),
	}

	for i := 0; i < capacity; i++ {
		w := &worker{pool}
		pool.workers[i] = w
		go w.run()
	}

	return pool
}

// SubmitTask adds a new task to the pool with an optional timeout duration.
// If the timeout duration is zero, the task will not have a timeout.
func (p *Pool) SubmitTask(task Task, timeout time.Duration) {
	if p.shutdown {
		return
	}

	if timeout == 0 {
		p.tasks <- task
		return
	}

	select {
	case p.tasks <- task:
	case <-time.After(timeout):
		runtime.Goexit()
	}
}

// SetCapacity sets the maximum number of workers that the pool can have.
func (p *Pool) SetCapacity(capacity int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if capacity < len(p.workers) {
		diff := len(p.workers) - capacity
		for i := 0; i < diff; i++ {
			p.workers[i].stop()
		}
		p.workers = p.workers[diff:]
	} else if capacity > len(p.workers) {
		diff := capacity - len(p.workers)
		for i := 0; i < diff; i++ {
			w := &worker{pool: p}
			p.workers = append(p.workers, w)
			go w.run()
		}
	}
}

// Shutdown shuts down the pool, preventing any new tasks from being submitted.
func (p *Pool) Shutdown() {
	p.shutdown = true
	close(p.tasks)
}

// worker.run waits for tasks to execute.
func (w *worker) run() {
	for task := range w.pool.tasks {
		task()
	}
}

// worker.stop stops the worker from waiting for tasks to execute.
func (w *worker) stop() {
	go func() {
		for range w.pool.tasks {
		}
	}()
}
