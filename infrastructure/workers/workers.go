package workers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/jrazmi/envoker/sdk/environment"
)

var (
	ErrWorkerShutdown  = errors.New("worker should shutdown")
	ErrPoolShutdown    = errors.New("pool should shutdown")
	ErrNoWorkAvailable = errors.New("no work available")
)

// Options represents the exportable worker configuration
type Options struct {
	Name         string        `env:"WORKER_NAME" default:"worker"`
	WorkerCount  int           `env:"WORKER_COUNT" default:"5"`
	PollInterval time.Duration `env:"WORKER_POLL_INTERVAL" default:"5s"`
	IdleInterval time.Duration `env:"WORKER_IDLE_INTERVAL" default:"30s"`
	MaxRetries   int           `env:"WORKER_MAX_RETRIES" default:"3"`
}

// options holds the internal runtime configuration
type options struct {
	name         string
	workerCount  int
	pollInterval time.Duration
	idleInterval time.Duration
	maxRetries   int
	middlewares  []Middleware
	metrics      WorkerPoolMetrics // Add metrics to options

	logger *slog.Logger
}

// Option is a function that configures the worker pool options
type Option func(*options)

// WorkerPool configures a worker pool to run tasks via processor interface
type WorkerPool[T Task] struct {
	// configuration
	processor    Processor[T]
	name         string
	workerCount  int
	pollInterval time.Duration
	idleInterval time.Duration
	maxRetries   int // Add this field
	log          *slog.Logger

	// work
	workFunc         WorkFunc // The final wrapped work function
	middlewares      []Middleware
	preProcessHooks  []PreProcessHook[T]
	postProcessHooks []PostProcessHook[T]
	metrics          WorkerPoolMetrics // Add metrics to options

	// control
	ctx        context.Context
	cancel     context.CancelFunc
	workers    sync.WaitGroup // Counter to track active workers
	stopMutex  sync.Mutex     // Ensures Stop() only runs once
	startMutex sync.Mutex     // Protects against multiple Start() calls
	running    bool           // Track if pool is running
	startTime  time.Time
	// Communication
	errors chan error // Tube for workers to report critical errors

}

// WithName sets the worker pool name
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithWorkerCount sets the number of workers
func WithWorkerCount(count int) Option {
	return func(o *options) {
		o.workerCount = count
	}
}

// WithPollInterval sets how often to poll for new work
func WithPollInterval(interval time.Duration) Option {
	return func(o *options) {
		o.pollInterval = interval
	}
}

// WithIdleInterval sets how long to wait when no work is available
func WithIdleInterval(interval time.Duration) Option {
	return func(o *options) {
		o.idleInterval = interval
	}
}

// WithLogger sets a custom logger
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithMaxRetries sets the maximum number of retry attempts
func WithMaxRetries(maxRetries int) Option {
	return func(o *options) {
		o.maxRetries = maxRetries
	}
}

// Now WithMiddleware works
func WithMiddleware(middlewares ...Middleware) Option {
	return func(o *options) {
		o.middlewares = append(o.middlewares, middlewares...)
	}
}

// WithMetrics sets a custom metrics collector
func WithMetrics(metrics WorkerPoolMetrics) Option {
	return func(o *options) {
		o.metrics = metrics
	}
}

// NewFromEnv creates a new worker pool using environment variables
func NewFromEnv[T Task](prefix string, processor Processor[T], opts ...Option) (*WorkerPool[T], error) {
	var cfg Options
	if err := environment.ParseEnvTags(prefix, &cfg); err != nil {
		return nil, fmt.Errorf("parsing worker config: %w", err)
	}

	return newWorkerPool(processor, cfg, opts...)
}

// New creates a new worker pool with the given name and processor
func NewWorkerPool[T Task](name string, workerCount int, processor Processor[T], opts ...Option) (*WorkerPool[T], error) {
	cfg := Options{
		Name:         name,
		WorkerCount:  workerCount,
		PollInterval: 1 * time.Second,
		IdleInterval: 30 * time.Second,
		MaxRetries:   3,
	}

	// Prepend the processor to the options
	pool, err := newWorkerPool(processor, cfg, opts...)
	if err != nil {
		return nil, fmt.Errorf("pool setup failure: %v", err)
	}
	return pool, nil
}

// newWorkerPool creates a new worker pool with given config and applies options
func newWorkerPool[T Task](processor Processor[T], cfg Options, opts ...Option) (*WorkerPool[T], error) {
	// Start with config-based options
	internalOpts := &options{
		name:         cfg.Name,
		workerCount:  cfg.WorkerCount,
		pollInterval: cfg.PollInterval,
		idleInterval: cfg.IdleInterval,
		maxRetries:   cfg.MaxRetries,
		metrics:      NewNoOpMetrics(), // Default to no-op metrics

	}

	// Apply functional options to override config
	for _, opt := range opts {
		opt(internalOpts)
	}

	// Set up default logger if none provided
	if internalOpts.logger == nil {
		internalOpts.logger = slog.Default()
	}

	// Ensure reasonable defaults
	if internalOpts.workerCount <= 0 {
		internalOpts.workerCount = 1
	}
	if internalOpts.pollInterval <= 0 {
		internalOpts.pollInterval = 5 * time.Second
	}
	if internalOpts.idleInterval <= 0 {
		internalOpts.idleInterval = 30 * time.Second
	}

	pool := &WorkerPool[T]{
		processor:    processor,
		name:         internalOpts.name,
		workerCount:  internalOpts.workerCount,
		pollInterval: internalOpts.pollInterval,
		idleInterval: internalOpts.idleInterval,
		log:          internalOpts.logger,
		maxRetries:   internalOpts.maxRetries,

		middlewares: internalOpts.middlewares,
		metrics:     internalOpts.metrics,
		errors:      make(chan error, internalOpts.workerCount),
	}
	pool.buildMiddlewareChain()

	return pool, nil

}

func (wp *WorkerPool[T]) Start(ctx context.Context) error {
	wp.startTime = time.Now()
	wp.startMutex.Lock()
	defer wp.startMutex.Unlock()
	wp.log.Info(strings.Repeat("=", 60))
	wp.log.InfoContext(ctx,
		"starting worker pool",
		"name", wp.name,
		"worker_count", wp.workerCount,
		"poll_interval", wp.pollInterval,
	)
	wp.log.Info(strings.Repeat("=", 60))
	wp.metrics.Start(ctx, wp.name)

	wp.ctx, wp.cancel = context.WithCancel(ctx)
	for i := 0; i < wp.workerCount; i++ {
		workerID := fmt.Sprintf("%s-worker-%d", wp.name, i+1)
		wp.workers.Add(1)
		go wp.worker(workerID)
	}
	wp.running = true
	wp.workers.Wait()

	close(wp.errors)
	wp.metrics.Stop(ctx)

	wp.log.InfoContext(ctx, "worker pool stopped", "name", wp.name, "total_runtime", time.Since(wp.startTime))
	wp.running = false
	return nil
}

// Stop gracefully stops the worker pool
func (wp *WorkerPool[T]) Stop() {
	wp.stopMutex.Lock()
	defer wp.stopMutex.Unlock()

	// Check if already stopped
	if !wp.running {
		wp.log.InfoContext(wp.ctx, "pool already stopped", "name", wp.name)
		return
	}

	wp.log.InfoContext(wp.ctx, "stopping worker pool", "name", wp.name)
	if wp.cancel != nil {
		wp.cancel()
		wp.running = false
	}
}

// infrastructure/workers/worker.go

func (wp *WorkerPool[T]) worker(workerID string) {
	defer wp.workers.Done()
	defer wp.metrics.RecordWorkerStopped()

	wp.log.InfoContext(wp.ctx, "worker started",
		"worker_id", workerID,
		"pool", wp.name)
	defer wp.log.InfoContext(context.Background(), "worker stopped",
		"worker_id", workerID,
		"pool", wp.name)

	wp.metrics.RecordWorkerStarted()

	// Adaptive polling configuration
	activePollInterval := wp.pollInterval
	idleInterval := wp.idleInterval
	currentInterval := 1 * time.Millisecond

	ticker := time.NewTicker(currentInterval)
	defer ticker.Stop()

	for {
		select {
		case <-wp.ctx.Done():
			wp.log.InfoContext(context.Background(), "worker received shutdown signal",
				"worker_id", workerID)
			return

		case <-ticker.C:
			// Wrap the entire work function with panic recovery
			err := wp.workWithPanicRecovery(wp.ctx, workerID)

			// Determine next polling interval based on result
			var newInterval time.Duration

			if err != nil {
				// Check for special error types
				if errors.Is(err, ErrWorkerShutdown) {
					wp.log.InfoContext(wp.ctx, "worker shutting down as requested",
						"worker_id", workerID)
					return
				}

				if errors.Is(err, ErrPoolShutdown) {
					wp.log.ErrorContext(wp.ctx, "worker requesting pool shutdown",
						"worker_id", workerID,
						"error", err)

					select {
					case wp.errors <- fmt.Errorf("worker %s: %w", workerID, err):
					default:
						wp.log.ErrorContext(wp.ctx, "error channel full, critical error not sent",
							"worker_id", workerID)
					}
					return
				}

				if errors.Is(err, ErrNoWorkAvailable) {
					newInterval = idleInterval
					if currentInterval != idleInterval {
						wp.log.InfoContext(wp.ctx, "no work available, switching to idle polling",
							"worker_id", workerID,
							"active_interval", activePollInterval,
							"idle_interval", idleInterval)
					}
				} else {
					newInterval = activePollInterval
					wp.log.ErrorContext(wp.ctx, "task processing error",
						"worker_id", workerID,
						"error", err)
				}
			} else {
				// Success! Work was processed
				newInterval = activePollInterval
				if currentInterval != activePollInterval {
					wp.log.InfoContext(wp.ctx, "work completed, switching to active polling",
						"worker_id", workerID,
						"active_interval", activePollInterval,
						"idle_interval", idleInterval)
				}
			}

			if newInterval != currentInterval {
				currentInterval = newInterval
				ticker.Reset(newInterval)
			}
		}
	}
}

// workWithPanicRecovery wraps the entire work function with panic recovery.  Checkout -> Process -> Complete/Fail
func (wp *WorkerPool[T]) workWithPanicRecovery(ctx context.Context, workerID string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			wp.log.ErrorContext(context.Background(), "panic recovered in worker",
				"worker_id", workerID,
				"panic", r,
				"stack_trace", string(stack))

			wp.metrics.RecordWorkerPanic()

			// Convert panic to error
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	// Call the configured work function (with any middleware applied)
	return wp.workFunc(ctx, workerID)
}

// work runs the process Checkout -> Process -> Complete/Fail. The process function is wrapped in its own panic recovery to distinguish between task panics and worker panics.
func (wp *WorkerPool[T]) work(ctx context.Context, workerID string) error {
	task, err := wp.processor.Checkout(ctx, workerID)
	if err != nil {
		if errors.Is(err, ErrNoWorkAvailable) {
			wp.metrics.RecordCheckoutError()
			return err
		}
		wp.metrics.RecordCheckoutError()
		return fmt.Errorf("checkout failed: %w", err)
	}
	wp.metrics.RecordTaskCheckedOut()

	// Track processing state
	var processErr error
	var processedTask T
	var duration time.Duration
	startTime := time.Now()

	defer func() {
		duration = time.Since(startTime)

		// Handle panic
		if r := recover(); r != nil {
			stack := debug.Stack()
			wp.log.ErrorContext(ctx, "panic recovered in task",
				"worker_id", workerID,
				"task_id", task.GetID(),
				"panic", r,
				"stack_trace", string(stack))

			wp.metrics.RecordWorkerPanic()
			processErr = fmt.Errorf("panic: %v", r)
		}

		// Run post-process hooks
		// For errors/panics: use original task
		// For success: use processedTask
		hookTask := processedTask
		if processErr != nil {
			hookTask = task // Use original task on any error
		}

		for _, hook := range wp.postProcessHooks {
			if err := hook(ctx, hookTask, processErr); err != nil {
				wp.log.ErrorContext(ctx, "post-process hook failed",
					"task_id", task.GetID(),
					"error", err)
			}
		}

		// Handle result (error or success)
		if processErr != nil {
			wp.metrics.RecordTaskFailed(duration)
			if failErr := wp.processor.Fail(ctx, task, processErr); failErr != nil {
				wp.log.ErrorContext(ctx, "failed to mark task as failed",
					"task_id", task.GetID(),
					"error", failErr)
			}
		} else {
			wp.metrics.RecordTaskCompleted(duration)
			if completeErr := wp.processor.Complete(ctx, processedTask, int(duration.Milliseconds())); completeErr != nil {
				wp.log.ErrorContext(ctx, "failed to mark task as complete",
					"task_id", task.GetID(),
					"error", completeErr)
			}
		}
	}()

	// Run pre-process hooks
	for _, hook := range wp.preProcessHooks {
		if err := hook(ctx, task); err != nil {
			wp.log.ErrorContext(ctx, "pre-process hook failed",
				"task_id", task.GetID(),
				"error", err)
		}
	}

	wp.log.InfoContext(ctx, "processing task",
		"worker_id", workerID,
		"task_id", task.GetID())

	// Process with retry logic
	processedTask, processErr = wp.processWithRetry(ctx, task)

	// Log the outcome
	if processErr != nil {
		wp.log.ErrorContext(ctx, "task processing failed",
			"worker_id", workerID,
			"task_id", task.GetID(),
			"error", processErr,
			"duration_ms", int(duration.Milliseconds()))
		return fmt.Errorf("task processing error: %w", processErr)
	}

	wp.log.InfoContext(ctx, "task completed",
		"worker_id", workerID,
		"task_id", task.GetID(),
		"duration_ms", int(duration.Milliseconds()))

	return nil
}

// processWithRetry handles retry logic with metrics (no panic recovery here)
func (wp *WorkerPool[T]) processWithRetry(ctx context.Context, task T) (T, error) {
	maxAttempts := wp.maxRetries
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	initialDelay := 1 * time.Second

	var lastErr error
	var processedTask T

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			wp.metrics.RecordRetryAttempt()
			wp.log.InfoContext(ctx, "retrying task",
				"task_id", task.GetID(),
				"attempt", attempt,
				"max_attempts", maxAttempts)

			// Exponential backoff
			delay := initialDelay * time.Duration(1<<(attempt-2))
			select {
			case <-ctx.Done():
				return processedTask, ctx.Err()
			case <-time.After(delay):
			}
		}

		// Just call processor.Process directly - panic recovery is at the top level
		processedTask, lastErr = wp.processor.Process(ctx, task)

		if lastErr == nil {
			if attempt > 1 {
				wp.metrics.RecordRetrySuccess()
			}
			return processedTask, nil
		}

		if ctx.Err() != nil {
			return processedTask, ctx.Err()
		}

		wp.log.ErrorContext(ctx, "task processing attempt failed",
			"task_id", task.GetID(),
			"attempt", attempt,
			"error", lastErr)
	}

	if maxAttempts > 1 {
		wp.metrics.RecordRetryExhausted()
	}

	return processedTask, fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}
func (wp *WorkerPool[T]) GetMetrics() MetricsSnapshot {
	return wp.metrics.GetSnapshot()
}
