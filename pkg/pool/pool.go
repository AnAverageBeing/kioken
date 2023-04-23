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
	capacity int
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
		tasks:    make(chan Task),
		workers:  make([]*worker, 0, capacity),
		capacity: capacity,
	}
	for i := 0; i < capacity; i++ {
		w := &worker{pool}
		pool.workers = append(pool.workers, w)
		go w.run()
	}
	return pool
}

// SubmitTask adds a new task to the pool with an optional timeout duration.
// If the timeout duration is zero, the task will not have a timeout.
func (p *Pool) SubmitTask(task Task, timeout time.Duration) {
	if timeout == 0 {
		p.tasks <- task
		return
	}

	if p.shutdown {
		return
	}
	select {
	case p.tasks <- task:
		go func() {
			select {
			case <-time.After(timeout):
				runtime.Goexit()
			default:
			}
		}()
	default:
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
			w.stop()
			p.workers = p.workers[:len(p.workers)-1]
		}
	} else if capacity > p.capacity {
		diff := capacity - p.capacity
		for i := 0; i < diff; i++ {
			w := &worker{pool: p}
			p.workers = append(p.workers, w)
			go w.run()
		}
	}
	p.capacity = capacity
}

// GrowPoolCapacity increases the capacity of the pool by the specified amount.
func (p *Pool) GrowPoolCapacity(delta int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	newCapacity := p.capacity + delta
	for i := p.capacity; i < newCapacity; i++ {
		w := &worker{pool: p}
		p.workers = append(p.workers, w)
		go w.run()
	}
	p.capacity = newCapacity
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
	close(w.pool.tasks)
}
