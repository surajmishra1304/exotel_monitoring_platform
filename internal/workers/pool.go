package workers

import (
	"sync"

	"exotel-monitoring-platform/internal/config"
	"exotel-monitoring-platform/internal/logger"
	"go.uber.org/zap"
)

// Pool is a bounded goroutine pool implemented with a semaphore channel.
// It prevents an unbounded number of goroutines from being spawned when
// hundreds of jobs become due simultaneously.
type Pool struct {
	sem  chan struct{}
	wg   sync.WaitGroup
}

var DefaultPool *Pool

// InitPool creates the application-wide worker pool.
func InitPool() {
	size := config.App.Scheduler.WorkerPoolSize
	if size <= 0 {
		size = 20
	}
	DefaultPool = NewPool(size)
	logger.Log.Info("worker pool initialised", zap.Int("size", size))
}

// NewPool creates a Pool with the given maximum concurrency.
func NewPool(maxWorkers int) *Pool {
	return &Pool{
		sem: make(chan struct{}, maxWorkers),
	}
}

// Submit enqueues fn for execution.
// It blocks if all workers are busy until a slot is free.
func (p *Pool) Submit(fn func()) {
	p.sem <- struct{}{} // acquire slot
	p.wg.Add(1)
	go func() {
		defer func() {
			<-p.sem // release slot
			p.wg.Done()
			// Recover from any panics inside a worker to avoid crashing the scheduler.
			if r := recover(); r != nil {
				logger.Log.Error("worker panic recovered", zap.Any("panic", r))
			}
		}()
		fn()
	}()
}

// Wait blocks until all submitted jobs complete.
func (p *Pool) Wait() {
	p.wg.Wait()
}
