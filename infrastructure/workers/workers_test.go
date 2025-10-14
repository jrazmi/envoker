package workers_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jrazmi/envoker/infrastructure/workers"
)

// ============================================================================
// Test Task Implementation
// ============================================================================
// For quiet tests (only errors)

type TestTask struct {
	ID        string
	Payload   string
	ShouldErr bool
	Delay     time.Duration
}

func (t TestTask) GetID() string {
	return t.ID
}

// ============================================================================
// Stubbed Processor Implementation
// ============================================================================

type StubProcessor struct {
	mu              sync.Mutex
	tasks           []TestTask
	checkoutCount   atomic.Int32
	processCount    atomic.Int32
	completeCount   atomic.Int32
	failCount       atomic.Int32
	checkoutErr     error
	processErr      error
	completeErr     error
	failErr         error
	processDelay    time.Duration
	infiniteTasks   bool
	taskIDCounter   atomic.Int32
	panicOnProcess  bool
	panicOnCheckout bool

	// Override functions - these can be set by tests
	checkoutFunc func(ctx context.Context, workerID string) (TestTask, error)
	processFunc  func(ctx context.Context, task TestTask) (TestTask, error)
}

func NewStubProcessor() *StubProcessor {
	return &StubProcessor{
		tasks: []TestTask{},
	}
}

func (p *StubProcessor) AddTask(task TestTask) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tasks = append(p.tasks, task)
}

func (p *StubProcessor) Checkout(ctx context.Context, workerID string) (TestTask, error) {

	// Use override if provided
	if p.checkoutFunc != nil {
		return p.checkoutFunc(ctx, workerID)
	}

	// Default implementation
	if p.panicOnCheckout {
		panic("checkout panic test")
	}

	p.checkoutCount.Add(1)

	if p.checkoutErr != nil {
		return TestTask{}, p.checkoutErr
	}

	if p.infiniteTasks {
		// Generate tasks infinitely for benchmarks
		id := p.taskIDCounter.Add(1)
		return TestTask{
			ID:      fmt.Sprintf("infinite-task-%d", id),
			Payload: "benchmark task",
		}, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.tasks) == 0 {
		return TestTask{}, workers.ErrNoWorkAvailable
	}

	task := p.tasks[0]
	p.tasks = p.tasks[1:]

	return task, nil
}

func (p *StubProcessor) Process(ctx context.Context, task TestTask) (TestTask, error) {
	// Use override if provided
	if p.processFunc != nil {
		return p.processFunc(ctx, task)
	}

	// Default implementation
	if p.panicOnProcess {
		panic("process panic test")
	}

	p.processCount.Add(1)

	// Simulate processing delay
	if p.processDelay > 0 {
		select {
		case <-ctx.Done():
			return task, ctx.Err()
		case <-time.After(p.processDelay):
		}
	}

	// Task-specific delay
	if task.Delay > 0 {
		select {
		case <-ctx.Done():
			return task, ctx.Err()
		case <-time.After(task.Delay):
		}
	}

	if p.processErr != nil {
		return task, p.processErr
	}

	if task.ShouldErr {
		return task, fmt.Errorf("task %s failed as requested", task.ID)
	}

	// Simulate transformation
	task.Payload = task.Payload + "-processed"
	return task, nil
}

func (p *StubProcessor) Complete(ctx context.Context, task TestTask, processingTimeMS int) error {
	p.completeCount.Add(1)
	return p.completeErr
}

func (p *StubProcessor) Fail(ctx context.Context, task TestTask, err error) error {
	p.failCount.Add(1)
	return p.failErr
}

// Helper methods for testing
func (p *StubProcessor) GetCheckoutCount() int32 {
	return p.checkoutCount.Load()
}

func (p *StubProcessor) GetProcessCount() int32 {
	return p.processCount.Load()
}

func (p *StubProcessor) GetCompleteCount() int32 {
	return p.completeCount.Load()
}

func (p *StubProcessor) GetFailCount() int32 {
	return p.failCount.Load()
}

func (p *StubProcessor) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.tasks = []TestTask{}
	p.checkoutCount.Store(0)
	p.processCount.Store(0)
	p.completeCount.Store(0)
	p.failCount.Store(0)
	p.checkoutErr = nil
	p.processErr = nil
	p.completeErr = nil
	p.failErr = nil
	p.processDelay = 0
	p.infiniteTasks = false
	p.taskIDCounter.Store(0)
	p.panicOnProcess = false
	p.panicOnCheckout = false
	p.checkoutFunc = nil
	p.processFunc = nil
}

// ============================================================================
// Tests
// ============================================================================

func TestWorkerPool_BasicFlow(t *testing.T) {
	processor := NewStubProcessor()

	// Add test tasks
	for i := 0; i < 5; i++ {
		processor.AddTask(TestTask{
			ID:      fmt.Sprintf("task-%d", i),
			Payload: fmt.Sprintf("payload-%d", i),
		})
	}

	pool, err := workers.NewWorkerPool("test-pool", 2, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))), workers.WithPollInterval(10*time.Millisecond),
		workers.WithIdleInterval(50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("pool returned error: %v", err)
		}
	case <-time.After(1 * time.Second):
		pool.Stop()
		<-done
	}

	if processor.GetProcessCount() != 5 {
		t.Errorf("expected 5 processes, got %d", processor.GetProcessCount())
	}
	if processor.GetCompleteCount() != 5 {
		t.Errorf("expected 5 completes, got %d", processor.GetCompleteCount())
	}
	if processor.GetFailCount() != 0 {
		t.Errorf("expected 0 failures, got %d", processor.GetFailCount())
	}
}

func TestWorkerPool_ErrorHandling(t *testing.T) {
	processor := NewStubProcessor()

	// Add tasks that will fail
	for i := 0; i < 3; i++ {
		processor.AddTask(TestTask{
			ID:        fmt.Sprintf("error-task-%d", i),
			Payload:   "will-fail",
			ShouldErr: true,
		})
	}

	pool, err := workers.NewWorkerPool("test-pool", 1, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
		workers.WithMaxRetries(1), // No retries for clearer test
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	// Let it run briefly
	time.Sleep(200 * time.Millisecond)
	pool.Stop()
	<-done

	// All tasks should have failed
	if processor.GetFailCount() != 3 {
		t.Errorf("expected 3 failures, got %d", processor.GetFailCount())
	}
	if processor.GetCompleteCount() != 0 {
		t.Errorf("expected 0 completes, got %d", processor.GetCompleteCount())
	}
}

func TestWorkerPool_Retry(t *testing.T) {
	processor := NewStubProcessor()

	// Add a task
	processor.AddTask(TestTask{
		ID:      "retry-task",
		Payload: "test",
	})

	// Task will fail first time, succeed second time
	attemptCount := atomic.Int32{}
	processor.processFunc = func(ctx context.Context, task TestTask) (TestTask, error) {
		processor.processCount.Add(1)
		attempt := attemptCount.Add(1)

		if attempt == 1 {
			return task, fmt.Errorf("temporary error")
		}

		task.Payload = task.Payload + "-processed"
		return task, nil
	}

	pool, err := workers.NewWorkerPool("test-pool", 1, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
		workers.WithMaxRetries(3),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	time.Sleep(2 * time.Second)
	pool.Stop()
	<-done

	// Add debug output in the test:
	t.Logf("MaxRetries setting: %d", 3) // Verify it's set
	t.Logf("Process attempts: %d", processor.GetProcessCount())

	// Should be:
	if processor.GetProcessCount() < 2 {
		t.Errorf("expected at least 2 process attempts (1 fail, 1 success), got %d", processor.GetProcessCount())
	}

	if processor.GetCompleteCount() != 1 {
		t.Errorf("expected 1 complete after retry, got %d", processor.GetCompleteCount())
	}

	if processor.GetFailCount() != 0 {
		t.Errorf("expected 0 failures (should retry and succeed), got %d", processor.GetFailCount())
	}
	// Check attemptCount, not errorCount:
	if attemptCount.Load() < 2 {
		t.Errorf("expected at least 2 process attempts, got %d", attemptCount.Load())
	}
}

func TestWorkerPool_PanicRecovery(t *testing.T) {
	processor := NewStubProcessor()

	// Only panic on first task
	processCount := atomic.Int32{}
	processor.processFunc = func(ctx context.Context, task TestTask) (TestTask, error) {
		processor.processCount.Add(1)
		count := processCount.Add(1)

		if count == 1 {
			panic("process panic test")
		}

		// Normal processing for subsequent tasks
		task.Payload = task.Payload + "-processed"
		return task, nil
	}

	// Add both tasks upfront
	processor.AddTask(TestTask{ID: "panic-task", Payload: "will-panic"})
	processor.AddTask(TestTask{ID: "normal-task", Payload: "should-work"})

	pool, err := workers.NewWorkerPool("test-pool", 1, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
		workers.WithLogger(slog.Default()),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	// Let it process
	time.Sleep(200 * time.Millisecond)

	// Disable panic for second task
	processor.panicOnProcess = false

	time.Sleep(200 * time.Millisecond)
	pool.Stop()
	<-done

	// First task should have failed due to panic
	// Second task should have succeeded
	if processor.GetFailCount() != 1 {
		t.Errorf("expected 1 failure from panic, got %d", processor.GetFailCount())
	}
	if processor.GetCompleteCount() != 1 {
		t.Errorf("expected 1 complete after panic recovery, got %d", processor.GetCompleteCount())
	}
}

func TestWorkerPool_GracefulShutdown(t *testing.T) {
	processor := NewStubProcessor()
	processor.processDelay = 100 * time.Millisecond

	// Add tasks that take time to process
	for i := 0; i < 5; i++ {
		processor.AddTask(TestTask{
			ID:      fmt.Sprintf("slow-task-%d", i),
			Payload: "slow",
		})
	}

	pool, err := workers.NewWorkerPool("test-pool", 2, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx := context.Background()

	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	// Let some processing start
	time.Sleep(150 * time.Millisecond)

	// Initiate graceful shutdown
	pool.Stop()

	// Wait for shutdown with timeout
	select {
	case <-done:
		// Good, shutdown completed
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown took too long")
	}

	// Some tasks should have been processed
	if processor.GetCheckoutCount() == 0 {
		t.Error("no tasks were checked out before shutdown")
	}
}

func TestWorkerPool_Hooks(t *testing.T) {
	processor := NewStubProcessor()
	processor.AddTask(TestTask{
		ID:      "hook-task",
		Payload: "test",
	})

	preHookCalled := atomic.Bool{}
	postHookCalled := atomic.Bool{}
	var postHookErr error

	pool, err := workers.NewWorkerPool("test-pool", 1, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	// Add hooks
	pool.AddPreProcessHooks(func(ctx context.Context, task TestTask) error {
		preHookCalled.Store(true)
		return nil
	})

	pool.AddPostProcessHooks(func(ctx context.Context, task TestTask, err error) error {
		postHookCalled.Store(true)
		postHookErr = err
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	time.Sleep(200 * time.Millisecond)
	pool.Stop()
	<-done

	if !preHookCalled.Load() {
		t.Error("pre-process hook was not called")
	}
	if !postHookCalled.Load() {
		t.Error("post-process hook was not called")
	}
	if postHookErr != nil {
		t.Errorf("post-process hook received unexpected error: %v", postHookErr)
	}
}

func TestWorkerPool_Metrics(t *testing.T) {
	processor := NewStubProcessor()

	// Add mix of successful and failing tasks
	for i := 0; i < 10; i++ {
		processor.AddTask(TestTask{
			ID:        fmt.Sprintf("metric-task-%d", i),
			Payload:   "test",
			ShouldErr: i%3 == 0, // Every 3rd task fails
		})
	}

	metrics := workers.NewInMemoryMetrics()
	pool, err := workers.NewWorkerPool("test-pool", 2, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
		workers.WithMetrics(metrics),
		workers.WithMaxRetries(1),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	time.Sleep(500 * time.Millisecond)
	pool.Stop()
	<-done

	snapshot := metrics.GetSnapshot()

	// Verify metrics
	if snapshot.TasksCompleted == 0 {
		t.Error("no tasks completed in metrics")
	}
	if snapshot.TasksFailed == 0 {
		t.Error("no tasks failed in metrics (expected some failures)")
	}
	if snapshot.WorkersStarted != 2 {
		t.Errorf("expected 2 workers started, got %d", snapshot.WorkersStarted)
	}
	if snapshot.WorkersActive != 0 {
		t.Errorf("expected 0 active workers after stop, got %d", snapshot.WorkersActive)
	}

	totalTasks := snapshot.TasksCompleted + snapshot.TasksFailed
	if totalTasks != 10 {
		t.Errorf("expected 10 total tasks processed, got %d", totalTasks)
	}
}

func TestWorkerPool_ConcurrentWorkers(t *testing.T) {
	processor := NewStubProcessor()
	processor.processDelay = 50 * time.Millisecond

	// Add enough tasks to ensure concurrent processing
	for i := 0; i < 20; i++ {
		processor.AddTask(TestTask{
			ID:      fmt.Sprintf("concurrent-task-%d", i),
			Payload: "test",
		})
	}

	workerCount := 5
	pool, err := workers.NewWorkerPool("test-pool", workerCount, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()
	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	// Let it process for a bit
	time.Sleep(1 * time.Second)
	pool.Stop()
	<-done
	duration := time.Since(start)

	processed := processor.GetProcessCount()

	// With concurrent workers, we should process faster than sequential
	// Expected time for sequential: 20 tasks * 50ms = 1000ms
	// With 5 workers: ~200ms + overhead
	expectedSequentialTime := time.Duration(processed) * processor.processDelay
	if duration > expectedSequentialTime/2 {
		t.Logf("Processing seems too slow for %d concurrent workers: %v vs expected max %v",
			workerCount, duration, expectedSequentialTime/2)
	}

	// Verify work was distributed
	if processed < 10 {
		t.Errorf("expected at least 10 tasks processed with concurrent workers, got %d", processed)
	}
}

func TestWorkerPool_IdlePolling(t *testing.T) {
	processor := NewStubProcessor()

	// Start with no tasks
	pool, err := workers.NewWorkerPool("test-pool", 1, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(20*time.Millisecond),
		workers.WithIdleInterval(100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	// Run with no work (should use idle interval)
	time.Sleep(150 * time.Millisecond)
	checkoutsNoWork := processor.GetCheckoutCount()

	// Add work
	processor.AddTask(TestTask{ID: "late-task", Payload: "test"})

	// Should switch to active polling
	time.Sleep(150 * time.Millisecond)

	pool.Stop()
	<-done

	// Should have attempted checkouts during idle period
	if checkoutsNoWork < 1 {
		t.Error("no checkout attempts during idle period")
	}

	// Task should have been processed
	if processor.GetCompleteCount() != 1 {
		t.Error("late task was not processed")
	}
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkWorkerPool_Throughput(b *testing.B) {
	benchmarks := []struct {
		name         string
		workerCount  int
		processDelay time.Duration
	}{
		{"1Worker_NoDelay", 1, 0},
		{"5Workers_NoDelay", 5, 0},
		{"10Workers_NoDelay", 10, 0},
		{"5Workers_1msDelay", 5, 1 * time.Millisecond},
		{"10Workers_1msDelay", 10, 1 * time.Millisecond},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			processor := NewStubProcessor()
			processor.infiniteTasks = true
			processor.processDelay = bm.processDelay

			pool, err := workers.NewWorkerPool("bench-pool", bm.workerCount, processor,
				workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
					Level: slog.LevelDebug,
				}))),
				workers.WithPollInterval(1*time.Millisecond),
				workers.WithMetrics(workers.NewNoOpMetrics()),
			)
			if err != nil {
				b.Fatalf("failed to create pool: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			b.ResetTimer()

			done := make(chan error, 1)
			go func() {
				done <- pool.Start(ctx)
			}()

			// Run for fixed duration
			time.Sleep(1 * time.Second)
			pool.Stop()
			<-done

			b.StopTimer()

			tasksProcessed := processor.GetProcessCount()
			b.ReportMetric(float64(tasksProcessed), "tasks/sec")
			b.ReportMetric(float64(tasksProcessed)/float64(bm.workerCount), "tasks/worker/sec")
		})
	}
}

func BenchmarkWorkerPool_MemoryAllocation(b *testing.B) {
	processor := NewStubProcessor()

	// Pre-populate with b.N tasks
	for i := 0; i < b.N; i++ {
		processor.AddTask(TestTask{
			ID:      fmt.Sprintf("bench-task-%d", i),
			Payload: "benchmark payload",
		})
	}

	pool, err := workers.NewWorkerPool("bench-pool", 5, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(1*time.Millisecond),
		workers.WithMetrics(workers.NewNoOpMetrics()),
	)
	if err != nil {
		b.Fatalf("failed to create pool: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	// Wait for all tasks to complete
	for processor.GetCompleteCount() < int32(b.N) {
		time.Sleep(10 * time.Millisecond)
	}

	pool.Stop()
	<-done
}

func BenchmarkWorkerPool_Latency(b *testing.B) {
	latencies := make([]time.Duration, 0, b.N)
	mu := sync.Mutex{}

	processor := NewStubProcessor()

	// Custom processor that measures latency
	processor.processFunc = func(ctx context.Context, task TestTask) (TestTask, error) {
		start := time.Now()
		processor.processCount.Add(1)

		// Simulate some work
		task.Payload = task.Payload + "-processed"

		latency := time.Since(start)

		mu.Lock()
		latencies = append(latencies, latency)
		mu.Unlock()

		return task, nil
	}

	// Add tasks
	for i := 0; i < b.N; i++ {
		processor.AddTask(TestTask{
			ID:      fmt.Sprintf("latency-task-%d", i),
			Payload: "test",
		})
	}

	pool, err := workers.NewWorkerPool("bench-pool", 5, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(1*time.Millisecond),
	)
	if err != nil {
		b.Fatalf("failed to create pool: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()

	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	// Wait for completion
	for processor.GetCompleteCount() < int32(b.N) {
		time.Sleep(10 * time.Millisecond)
	}

	pool.Stop()
	<-done

	b.StopTimer()

	// Calculate percentiles
	if len(latencies) > 0 {
		avgLatency := time.Duration(0)
		for _, l := range latencies {
			avgLatency += l
		}
		avgLatency /= time.Duration(len(latencies))

		b.ReportMetric(float64(avgLatency.Nanoseconds()), "ns/op")
	}
}
