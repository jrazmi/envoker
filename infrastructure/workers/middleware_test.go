package workers_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jrazmi/envoker/infrastructure/workers"
)

// Test middleware that counts invocations
func testCounterMiddleware(counter *atomic.Int32) workers.Middleware {
	return func(next workers.WorkFunc) workers.WorkFunc {
		return func(ctx context.Context, workerID string) error {
			counter.Add(1)
			return next(ctx, workerID)
		}
	}
}

// Test middleware that can inject errors
func testErrorMiddleware(shouldError *atomic.Bool, errorMsg string) workers.Middleware {
	return func(next workers.WorkFunc) workers.WorkFunc {
		return func(ctx context.Context, workerID string) error {
			if shouldError.Load() {
				return fmt.Errorf("test error mid: %s", errorMsg)
			}
			return next(ctx, workerID)
		}
	}
}

func TestMiddleware_ExecutionOrder(t *testing.T) {
	processor := NewStubProcessor()
	processor.AddTask(TestTask{ID: "test-1", Payload: "test"})

	executionOrder := make([]string, 0)
	var mu atomic.Pointer[[]string]
	mu.Store(&executionOrder)

	// Create middlewares that record their execution
	middleware1 := func(next workers.WorkFunc) workers.WorkFunc {
		return func(ctx context.Context, workerID string) error {
			order := mu.Load()
			*order = append(*order, "middleware1-before")
			err := next(ctx, workerID)
			*order = append(*order, "middleware1-after")
			return err
		}
	}

	middleware2 := func(next workers.WorkFunc) workers.WorkFunc {
		return func(ctx context.Context, workerID string) error {
			order := mu.Load()
			*order = append(*order, "middleware2-before")
			err := next(ctx, workerID)
			*order = append(*order, "middleware2-after")
			return err
		}
	}

	pool, err := workers.NewWorkerPool("test-pool", 1, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
		workers.WithMiddleware(middleware1, middleware2),
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

	time.Sleep(200 * time.Millisecond)
	pool.Stop()
	<-done

	// Verify execution order (middleware1 wraps middleware2)
	expected := []string{
		"middleware1-before",
		"middleware2-before",
		"middleware2-after",
		"middleware1-after",
	}

	if len(executionOrder) < len(expected) {
		t.Fatalf("expected at least %d execution steps, got %d", len(expected), len(executionOrder))
	}

	// Check the first execution sequence
	for i, exp := range expected {
		if executionOrder[i] != exp {
			t.Errorf("execution order mismatch at index %d: expected %s, got %s",
				i, exp, executionOrder[i])
		}
	}
}

func TestMiddleware_ConsecutiveErrorShutdown(t *testing.T) {
	processor := NewStubProcessor()

	// Add tasks that will fail
	for i := 0; i < 10; i++ {
		processor.AddTask(TestTask{
			ID:        fmt.Sprintf("error-%d", i),
			Payload:   "test",
			ShouldErr: true, // All tasks fail
		})
	}

	pool, err := workers.NewWorkerPool("test-pool", 1, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
		workers.WithMaxRetries(1), // No retries
		workers.WithMiddleware(workers.ConsecutiveErrorShutdown(3)),
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

	// Wait for shutdown
	select {
	case <-done:
		// Worker should have shut down after 3 consecutive errors
	case <-time.After(1 * time.Second):
		pool.Stop()
		<-done
	}

	// Should have processed more than 3 but not all 10
	processed := processor.GetProcessCount()
	if processed <= 3 {
		t.Errorf("expected more than 3 tasks processed before shutdown, got %d", processed)
	}
	if processed >= 10 {
		t.Errorf("worker should have shut down before processing all tasks, processed %d", processed)
	}
}

func TestMiddleware_ChainedMiddleware(t *testing.T) {
	processor := NewStubProcessor()
	processor.AddTask(TestTask{ID: "chain-test", Payload: "test"})

	counter1 := atomic.Int32{}
	counter2 := atomic.Int32{}
	counter3 := atomic.Int32{}

	pool, err := workers.NewWorkerPool("test-pool", 1, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
		workers.WithMiddleware(
			testCounterMiddleware(&counter1),
			testCounterMiddleware(&counter2),
			testCounterMiddleware(&counter3),
		),
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

	time.Sleep(200 * time.Millisecond)
	pool.Stop()
	<-done

	// All middlewares should have been called
	if counter1.Load() == 0 {
		t.Error("middleware 1 was not called")
	}
	if counter2.Load() == 0 {
		t.Error("middleware 2 was not called")
	}
	if counter3.Load() == 0 {
		t.Error("middleware 3 was not called")
	}

	// They should be called the same number of times
	if counter1.Load() != counter2.Load() || counter2.Load() != counter3.Load() {
		t.Errorf("middleware call counts differ: %d, %d, %d",
			counter1.Load(), counter2.Load(), counter3.Load())
	}
}

func TestMiddleware_ErrorPropagation(t *testing.T) {
	processor := NewStubProcessor()
	processor.AddTask(TestTask{ID: "error-prop", Payload: "test"})

	shouldError := atomic.Bool{}
	shouldError.Store(true)

	errorsSeen := atomic.Int32{}

	// Middleware that counts errors it sees
	errorCountingMiddleware := func(next workers.WorkFunc) workers.WorkFunc {
		return func(ctx context.Context, workerID string) error {
			err := next(ctx, workerID)
			if err != nil && !errors.Is(err, workers.ErrNoWorkAvailable) {
				errorsSeen.Add(1)
			}
			return err
		}
	}

	pool, err := workers.NewWorkerPool("test-pool", 1, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
		workers.WithMiddleware(
			errorCountingMiddleware,
			testErrorMiddleware(&shouldError, "injected error"),
		),
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

	time.Sleep(200 * time.Millisecond)

	// Stop injecting errors
	shouldError.Store(false)

	time.Sleep(200 * time.Millisecond)
	pool.Stop()
	<-done

	// Outer middleware should have seen the errors
	if errorsSeen.Load() == 0 {
		t.Error("outer middleware did not see errors from inner middleware")
	}
}

func TestMiddleware_PanicRecovery(t *testing.T) {
	processor := NewStubProcessor()
	processor.AddTask(TestTask{ID: "panic-test", Payload: "test"})

	panicMiddleware := func(next workers.WorkFunc) workers.WorkFunc {
		return func(ctx context.Context, workerID string) error {
			panic("middleware panic!")
		}
	}

	pool, err := workers.NewWorkerPool("test-pool", 1, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelError,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
		workers.WithLogger(slog.Default()),
		workers.WithMiddleware(panicMiddleware),
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

	time.Sleep(300 * time.Millisecond)
	pool.Stop()

	select {
	case <-done:
		// Pool should have recovered from panic and stopped gracefully
	case <-time.After(2 * time.Second):
		t.Fatal("pool did not stop after middleware panic")
	}
}

// Benchmark middleware overhead
func BenchmarkMiddleware_Overhead(b *testing.B) {
	benchmarks := []struct {
		name            string
		middlewareCount int
	}{
		{"NoMiddleware", 0},
		{"1Middleware", 1},
		{"3Middleware", 3},
		{"5Middleware", 5},
		{"10Middleware", 10},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			processor := NewStubProcessor()

			// Add b.N tasks
			for i := 0; i < b.N; i++ {
				processor.AddTask(TestTask{
					ID:      fmt.Sprintf("bench-%d", i),
					Payload: "test",
				})
			}

			// Create middleware chain
			middlewares := make([]workers.Middleware, bm.middlewareCount)
			for i := 0; i < bm.middlewareCount; i++ {
				counter := &atomic.Int32{} // Each middleware gets its own counter
				middlewares[i] = testCounterMiddleware(counter)
			}

			pool, err := workers.NewWorkerPool("bench-pool", 1, processor,
				workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
					Level: slog.LevelError,
				}))),
				workers.WithPollInterval(1*time.Millisecond),
				workers.WithMiddleware(middlewares...),
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

			// Wait for all tasks
			for processor.GetProcessCount() < int32(b.N) {
				time.Sleep(5 * time.Millisecond)
			}

			pool.Stop()
			<-done

			b.StopTimer()
		})
	}
}
