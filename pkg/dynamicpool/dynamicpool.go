package dynamicpool

import (
	"sync"
)

// DynamicPool is a dynamically-sized pool of goroutines.
type DynamicPool struct {
	maxWorkers int
	tasks      chan func()
	wg         sync.WaitGroup
}

// NewDynamicPool creates a new dynamic pool with the specified maximum number of workers.
func NewDynamicPool(maxWorkers int) *DynamicPool {
	return &DynamicPool{
		maxWorkers: maxWorkers,
		tasks:      make(chan func()),
	}
}

// Start starts the dynamic pool.
func (p *DynamicPool) Start() {
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for task := range p.tasks {
				task()
			}
		}()
	}
}

// Stop stops the dynamic pool.
func (p *DynamicPool) Stop() {
	close(p.tasks)
	p.wg.Wait()
}

// Submit submits a task to the dynamic pool.
func (p *DynamicPool) Submit(task func()) {
	select {
	case p.tasks <- task:
	default:
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for task := range p.tasks {
				task()
			}
		}()
	}
}
