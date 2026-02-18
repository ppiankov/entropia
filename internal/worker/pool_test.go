package worker

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockResult implements Result
type mockResult struct {
	err error
}

func (r *mockResult) GetError() error {
	return r.err
}

// mockJob implements Job
type mockJob struct {
	duration  time.Duration
	shouldErr bool
	executed  *int32 // atomic counter
}

func (j *mockJob) Execute(ctx context.Context) Result {
	if j.executed != nil {
		atomic.AddInt32(j.executed, 1)
	}
	if j.duration > 0 {
		select {
		case <-time.After(j.duration):
		case <-ctx.Done():
			return &mockResult{err: ctx.Err()}
		}
	}
	if j.shouldErr {
		return &mockResult{err: errors.New("job error")}
	}
	return &mockResult{err: nil}
}

func TestNewPool(t *testing.T) {
	p1 := NewPool(5)
	if p1.workers != 5 {
		t.Errorf("expected 5 workers, got %d", p1.workers)
	}

	p2 := NewPool(0)
	if p2.workers != 1 {
		t.Errorf("expected default 1 worker for 0 input, got %d", p2.workers)
	}

	p3 := NewPool(-1)
	if p3.workers != 1 {
		t.Errorf("expected default 1 worker for negative input, got %d", p3.workers)
	}
}

func TestPool_Execution(t *testing.T) {
	pool := NewPool(2)
	pool.Start()

	var executed int32
	count := 10

	for i := 0; i < count; i++ {
		pool.Submit(&mockJob{executed: &executed})
	}

	results := pool.Wait()

	if len(results) != count {
		t.Errorf("expected %d results, got %d", count, len(results))
	}

	if atomic.LoadInt32(&executed) != int32(count) {
		t.Errorf("expected %d executed jobs, got %d", count, executed)
	}
}

// concurrencyJob tracks max concurrent executions
type concurrencyJob struct {
	start    func()
	end      func()
	duration time.Duration
}

func (j *concurrencyJob) Execute(ctx context.Context) Result {
	if j.start != nil {
		j.start()
	}
	time.Sleep(j.duration)
	if j.end != nil {
		j.end()
	}
	return &mockResult{}
}

func TestPool_Concurrency(t *testing.T) {
	workers := 10
	pool := NewPool(workers)
	pool.Start()

	var current int32
	var maxConcurrent int32
	var completed int32
	var mu sync.Mutex

	totalJobs := 50

	for i := 0; i < totalJobs; i++ {
		pool.Submit(&concurrencyJob{
			start: func() {
				curr := atomic.AddInt32(&current, 1)
				mu.Lock()
				if curr > maxConcurrent {
					maxConcurrent = curr
				}
				mu.Unlock()
			},
			end: func() {
				atomic.AddInt32(&current, -1)
				atomic.AddInt32(&completed, 1)
			},
			duration: 10 * time.Millisecond,
		})
	}

	pool.Wait()

	if atomic.LoadInt32(&completed) != int32(totalJobs) {
		t.Errorf("expected %d completed jobs, got %d", totalJobs, completed)
	}

	mu.Lock()
	max := maxConcurrent
	mu.Unlock()

	if max > int32(workers) {
		t.Errorf("max concurrency %d exceeded workers %d", max, workers)
	}

	if max <= 1 {
		t.Logf("Warning: max concurrency was %d, expected > 1", max)
	}
}

func TestPool_ErrorHandling(t *testing.T) {
	pool := NewPool(2)
	pool.Start()

	pool.Submit(&mockJob{shouldErr: true})
	pool.Submit(&mockJob{shouldErr: false})

	results := pool.Wait()
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	errors := 0
	for _, res := range results {
		if res.GetError() != nil {
			errors++
		}
	}

	if errors != 1 {
		t.Errorf("expected 1 error, got %d", errors)
	}
}

func TestResultCollector(t *testing.T) {
	c := NewResultCollector()
	c.Add(&mockResult{})
	c.Add(&mockResult{err: errors.New("err")})

	res := c.Results()
	if len(res) != 2 {
		t.Errorf("expected 2 results, got %d", len(res))
	}
}

func TestPool_SubmitAfterShutdown(t *testing.T) {
	pool := NewPool(2)
	pool.Start()
	pool.Shutdown()

	// Submit after shutdown should not panic or block
	done := make(chan struct{})
	go func() {
		pool.Submit(&mockJob{})
		close(done)
	}()

	select {
	case <-done:
		// success — Submit returned without blocking
	case <-time.After(1 * time.Second):
		t.Fatal("Submit after shutdown blocked")
	}
}

func TestPool_Shutdown(t *testing.T) {
	pool := NewPool(2)
	pool.Start()

	// Use a channel to synchronize start of job
	started := make(chan struct{})

	pool.Submit(&concurrencyJob{
		start: func() {
			close(started)
		},
		duration: 200 * time.Millisecond,
	})

	// Wait for job to start
	<-started

	// Shutdown immediately
	pool.Shutdown()

	// Ensure Shutdown returns and closes results
	done := make(chan struct{})
	go func() {
		for range pool.results {
		}
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(1 * time.Second):
		t.Fatal("Shutdown timed out")
	}
}
