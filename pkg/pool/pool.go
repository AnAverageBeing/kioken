package pool

import (
	"sync"
)

// Task is a function that can be executed by the goroutines in the pool.
type Task func()

// Pool is a dynamically growing goroutine pool.
type Pool struct {
	tasks    chan Task
	workers  []*worker
	capacity int
	mutex    sync.Mutex
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

// SubmitTask adds a new task to the pool.
func (p *Pool) SubmitTask(task Task) {
	p.tasks <- task
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
