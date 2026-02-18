package worker

import (
	"context"
	"sync"
)

// Job represents a unit of work to be executed
type Job interface {
	Execute(ctx context.Context) Result
}

// Result represents the result of a job execution
type Result interface {
	GetError() error
}

// Pool manages a pool of workers that execute jobs concurrently
type Pool struct {
	workers    int
	jobQueue   chan Job
	results    chan Result
	wg         sync.WaitGroup
	ctx        context.Context
	cancelFunc context.CancelFunc
	closeOnce  sync.Once
}

// NewPool creates a new worker pool with the specified number of workers
func NewPool(workers int) *Pool {
	if workers <= 0 {
		workers = 1
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Pool{
		workers:    workers,
		jobQueue:   make(chan Job, workers*2), // Buffered to prevent blocking
		results:    make(chan Result, workers*2),
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// Start starts the worker pool
func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker is the worker goroutine that processes jobs
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobQueue:
			if !ok {
				return
			}
			result := job.Execute(p.ctx)
			select {
			case p.results <- result:
			case <-p.ctx.Done():
				return
			}
		}
	}
}

// Submit submits a job to the pool for execution
func (p *Pool) Submit(job Job) {
	select {
	case <-p.ctx.Done():
		return
	case p.jobQueue <- job:
	}
}

// Wait waits for all jobs to complete and returns the results
func (p *Pool) Wait() []Result {
	// Close job queue to signal workers to exit when done
	close(p.jobQueue)

	// Use a goroutine to wait for workers and close results
	go func() {
		p.wg.Wait()
		p.closeResults()
	}()

	// Collect all results
	var results []Result
	for result := range p.results {
		results = append(results, result)
	}

	return results
}

// Shutdown shuts down the worker pool immediately
func (p *Pool) Shutdown() {
	p.cancelFunc()
	p.wg.Wait()
	p.closeResults()
}

func (p *Pool) closeResults() {
	p.closeOnce.Do(func() {
		close(p.results)
	})
}

// ResultCollector provides a safer way to collect results as they arrive
type ResultCollector struct {
	results []Result
	mu      sync.Mutex
}

// NewResultCollector creates a new result collector
func NewResultCollector() *ResultCollector {
	return &ResultCollector{
		results: make([]Result, 0),
	}
}

// Add adds a result to the collector (thread-safe)
func (c *ResultCollector) Add(result Result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = append(c.results, result)
}

// Results returns all collected results
func (c *ResultCollector) Results() []Result {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.results
}