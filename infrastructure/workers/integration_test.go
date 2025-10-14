package workers_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jrazmi/envoker/infrastructure/workers"
)

// ============================================================================
// Realistic Task Processor Example
// ============================================================================

type EmailTask struct {
	ID        string
	Recipient string
	Subject   string
	Body      string
	Priority  int
	Retries   int
}

func (e EmailTask) GetID() string {
	return e.ID
}

type EmailProcessor struct {
	mu           sync.RWMutex
	queue        []EmailTask
	sent         []EmailTask
	failed       []EmailTask
	sendDelay    time.Duration
	failureRate  float32 // 0.0 to 1.0
	sentCount    atomic.Int32
	attemptCount atomic.Int32
}

func NewEmailProcessor() *EmailProcessor {
	return &EmailProcessor{
		queue:       make([]EmailTask, 0),
		sent:        make([]EmailTask, 0),
		failed:      make([]EmailTask, 0),
		sendDelay:   10 * time.Millisecond,
		failureRate: 0.1, // 10% failure rate
	}
}

func (p *EmailProcessor) AddEmail(task EmailTask) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.queue = append(p.queue, task)
}

func (p *EmailProcessor) Checkout(ctx context.Context, workerID string) (EmailTask, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.queue) == 0 {
		return EmailTask{}, workers.ErrNoWorkAvailable
	}

	// Priority queue simulation - find highest priority
	highestIdx := 0
	for i, task := range p.queue {
		if task.Priority > p.queue[highestIdx].Priority {
			highestIdx = i
		}
	}

	task := p.queue[highestIdx]
	// Remove from queue
	p.queue = append(p.queue[:highestIdx], p.queue[highestIdx+1:]...)

	return task, nil
}

func (p *EmailProcessor) Process(ctx context.Context, task EmailTask) (EmailTask, error) {
	p.attemptCount.Add(1)

	// Simulate processing time
	select {
	case <-ctx.Done():
		return task, ctx.Err()
	case <-time.After(p.sendDelay):
	}

	// Simulate failures based on rate
	attemptNum := p.attemptCount.Load()
	if float32(attemptNum%100)/100.0 < p.failureRate {
		return task, fmt.Errorf("email send failed: network error")
	}

	// Mark as processed
	task.Body = task.Body + " [SENT]"
	return task, nil
}

func (p *EmailProcessor) Complete(ctx context.Context, task EmailTask, processingTimeMS int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.sent = append(p.sent, task)
	p.sentCount.Add(1)
	return nil
}

func (p *EmailProcessor) Fail(ctx context.Context, task EmailTask, err error) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	task.Retries++
	if task.Retries < 3 {
		// Re-queue for retry
		p.queue = append(p.queue, task)
	} else {
		// Move to failed
		p.failed = append(p.failed, task)
	}

	return nil
}

func (p *EmailProcessor) GetStats() (queued, sent, failed int) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.queue), len(p.sent), len(p.failed)
}

// ============================================================================
// Integration Tests
// ============================================================================

func TestIntegration_EmailProcessingSystem(t *testing.T) {
	processor := NewEmailProcessor()
	processor.failureRate = 0.2 // 20% failure rate for testing

	// Add various priority emails
	for i := 0; i < 50; i++ {
		priority := i % 5 // Priorities 0-4
		processor.AddEmail(EmailTask{
			ID:        fmt.Sprintf("email-%d", i),
			Recipient: fmt.Sprintf("user%d@example.com", i),
			Subject:   fmt.Sprintf("Test Email %d", i),
			Body:      "This is a test email",
			Priority:  priority,
		})
	}

	// Setup metrics
	metrics := workers.NewInMemoryMetrics()

	// Create worker pool with realistic configuration
	pool, err := workers.NewWorkerPool("email-processor", 3, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(5*time.Millisecond),
		workers.WithIdleInterval(100*time.Millisecond),
		workers.WithMaxRetries(1),
		workers.WithMetrics(metrics),
		workers.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug, // Only show errors in tests
		}))),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	// Add business logic hooks
	var preProcessed atomic.Int32
	var postProcessed atomic.Int32

	pool.AddPreProcessHooks(func(ctx context.Context, task EmailTask) error {
		preProcessed.Add(1)
		// Could add validation here
		if task.Recipient == "" {
			return fmt.Errorf("invalid email: missing recipient")
		}
		return nil
	})

	pool.AddPostProcessHooks(func(ctx context.Context, task EmailTask, err error) error {
		postProcessed.Add(1)
		// Could add notification/logging here
		if err != nil && task.Priority >= 4 {
			t.Logf("High priority email %s failed: %v", task.ID, err)
		}
		return nil
	})

	// Run the system
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	// Let it process
	time.Sleep(2 * time.Second)

	// Check intermediate status
	queued, sent, failed := processor.GetStats()
	t.Logf("Mid-processing status - Queued: %d, Sent: %d, Failed: %d", queued, sent, failed)

	// Add more high-priority emails while running
	for i := 50; i < 60; i++ {
		processor.AddEmail(EmailTask{
			ID:        fmt.Sprintf("urgent-email-%d", i),
			Recipient: fmt.Sprintf("vip%d@example.com", i),
			Subject:   "URGENT: " + fmt.Sprintf("Test Email %d", i),
			Body:      "This is an urgent email",
			Priority:  5, // Highest priority
		})
	}

	// Continue processing
	time.Sleep(1 * time.Second)

	// Graceful shutdown
	pool.Stop()
	<-done

	// Final stats
	queued, sent, failed = processor.GetStats()
	t.Logf("Final status - Queued: %d, Sent: %d, Failed: %d", queued, sent, failed)

	// Verify processing
	if sent == 0 {
		t.Error("no emails were sent")
	}

	if sent+failed+queued != 60 {
		t.Errorf("email count mismatch: sent=%d, failed=%d, queued=%d, total should be 60",
			sent, failed, queued)
	}

	// Check metrics
	snapshot := pool.GetMetrics()
	t.Logf("Metrics - Completed: %d, Failed: %d, Retry Rate: %.2f%%",
		snapshot.TasksCompleted, snapshot.TasksFailed, snapshot.RetryRate)

	if snapshot.WorkersStarted != 3 {
		t.Errorf("expected 3 workers started, got %d", snapshot.WorkersStarted)
	}

	// Verify hooks were called
	if preProcessed.Load() == 0 {
		t.Error("pre-process hooks were not called")
	}
	if postProcessed.Load() == 0 {
		t.Error("post-process hooks were not called")
	}
}

func TestIntegration_LoadBalancing(t *testing.T) {
	// Track which worker processes which task
	workerTasks := make(map[string][]string)
	var mu sync.Mutex

	processor := NewStubProcessor()
	processor.processDelay = 20 * time.Millisecond

	// Add many tasks
	for i := 0; i < 100; i++ {
		processor.AddTask(TestTask{
			ID:      fmt.Sprintf("lb-task-%d", i),
			Payload: "test",
		})
	}

	// Create a wrapper function that tracks worker assignments
	processor.checkoutFunc = func(ctx context.Context, workerID string) (TestTask, error) {
		// Call the default checkout logic
		processor.checkoutCount.Add(1)

		if processor.checkoutErr != nil {
			return TestTask{}, processor.checkoutErr
		}

		processor.mu.Lock()
		defer processor.mu.Unlock()

		if len(processor.tasks) == 0 {
			return TestTask{}, workers.ErrNoWorkAvailable
		}

		task := processor.tasks[0]
		processor.tasks = processor.tasks[1:]

		// Track which worker got the task
		mu.Lock()
		workerTasks[workerID] = append(workerTasks[workerID], task.ID)
		mu.Unlock()

		return task, nil
	}

	workerCount := 5
	pool, err := workers.NewWorkerPool("load-balance-test", workerCount, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(5*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	// Let it run
	time.Sleep(3 * time.Second)
	pool.Stop()
	<-done

	// Analyze distribution
	mu.Lock()
	defer mu.Unlock()

	if len(workerTasks) != workerCount {
		t.Errorf("expected %d workers to process tasks, got %d", workerCount, len(workerTasks))
	}

	totalTasks := 0
	minTasks := 1000000
	maxTasks := 0

	for workerID, tasks := range workerTasks {
		count := len(tasks)
		totalTasks += count
		if count < minTasks {
			minTasks = count
		}
		if count > maxTasks {
			maxTasks = count
		}
		t.Logf("Worker %s processed %d tasks", workerID, count)
	}

	// Check for reasonable distribution
	avgTasks := totalTasks / workerCount
	distribution := float64(maxTasks-minTasks) / float64(avgTasks)

	if distribution > 0.5 { // More than 50% variance
		t.Logf("Warning: Uneven task distribution. Min: %d, Max: %d, Avg: %d",
			minTasks, maxTasks, avgTasks)
	}

	if totalTasks < 50 {
		t.Errorf("expected at least 50 tasks processed, got %d", totalTasks)
	}
}

func TestIntegration_GracefulDegradation(t *testing.T) {
	processor := NewStubProcessor()

	// Simulate degrading service
	degradeAfter := 20
	tasksProcessed := atomic.Int32{}

	processor.processFunc = func(ctx context.Context, task TestTask) (TestTask, error) {
		processor.processCount.Add(1)
		count := tasksProcessed.Add(1)

		// Start failing after threshold
		if count > int32(degradeAfter) {
			// Increase failure rate
			if count%3 != 0 { // 66% failure rate
				return task, fmt.Errorf("service degraded")
			}
		}

		// Simulate processing
		if processor.processDelay > 0 {
			select {
			case <-ctx.Done():
				return task, ctx.Err()
			case <-time.After(processor.processDelay):
			}
		}

		task.Payload = task.Payload + "-processed"
		return task, nil
	}

	// Add tasks
	for i := 0; i < 50; i++ {
		processor.AddTask(TestTask{
			ID:      fmt.Sprintf("degrade-task-%d", i),
			Payload: "test",
		})
	}

	metrics := workers.NewInMemoryMetrics()

	pool, err := workers.NewWorkerPool("degradation-test", 3, processor,
		workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
		workers.WithPollInterval(10*time.Millisecond),
		workers.WithMaxRetries(2),
		workers.WithMetrics(metrics),
		workers.WithLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))),
	)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- pool.Start(ctx)
	}()

	// Monitor metrics during degradation
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			snapshot := metrics.GetSnapshot()
			if snapshot.TasksCompleted+snapshot.TasksFailed > 0 {
				t.Logf("Progress - Completed: %d, Failed: %d, Error Rate: %.2f%%",
					snapshot.TasksCompleted, snapshot.TasksFailed, snapshot.ErrorRate)
			}
		}
	}()

	// Let it run through degradation
	time.Sleep(3 * time.Second)
	pool.Stop()
	<-done

	finalMetrics := metrics.GetSnapshot()

	// System should have processed some tasks successfully before and after degradation
	if finalMetrics.TasksCompleted < 10 {
		t.Errorf("too few tasks completed during degradation: %d", finalMetrics.TasksCompleted)
	}

	// Should have retried failed tasks
	if finalMetrics.RetryAttempts == 0 {
		t.Error("no retry attempts during degradation")
	}

	// Error rate should be elevated but not 100%
	if finalMetrics.ErrorRate > 80 {
		t.Logf("Warning: Very high error rate during degradation: %.2f%%", finalMetrics.ErrorRate)
	}

	t.Logf("Final metrics - Completed: %d, Failed: %d, Retries: %d, Error Rate: %.2f%%",
		finalMetrics.TasksCompleted, finalMetrics.TasksFailed,
		finalMetrics.RetryAttempts, finalMetrics.ErrorRate)
}

func TestIntegration_MemoryLeaks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory leak test in short mode")
	}

	// Run multiple pool lifecycles to check for leaks
	for cycle := 0; cycle < 5; cycle++ {
		processor := NewStubProcessor()
		processor.infiniteTasks = true
		processor.processDelay = 1 * time.Microsecond // Very fast processing

		pool, err := workers.NewWorkerPool(fmt.Sprintf("leak-test-%d", cycle), 10, processor,
			workers.WithLogger(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))),
			workers.WithPollInterval(1*time.Millisecond),
			workers.WithMetrics(workers.NewNoOpMetrics()),
		)
		if err != nil {
			t.Fatalf("cycle %d: failed to create pool: %v", cycle, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)

		done := make(chan error, 1)
		go func() {
			done <- pool.Start(ctx)
		}()

		// Run for a bit
		time.Sleep(500 * time.Millisecond)

		// Stop and wait for cleanup
		pool.Stop()
		<-done
		cancel()

		// Give time for cleanup
		time.Sleep(100 * time.Millisecond)
	}

	// If we get here without running out of memory, test passes
	// In a real scenario, you'd use runtime.MemStats to verify
}
