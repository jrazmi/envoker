// infrastructure/workers/metrics.go
package workers

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// WorkerPoolMetrics defines the interface for collecting pool orchestration metrics
type WorkerPoolMetrics interface {
	// Worker lifecycle
	RecordWorkerStarted()
	RecordWorkerStopped()
	RecordWorkerPanic()

	// Task flow
	RecordTaskCheckedOut()
	RecordTaskCompleted(duration time.Duration)
	RecordTaskFailed(duration time.Duration)
	RecordCheckoutError() // No work available or checkout failed

	// Retry metrics
	RecordRetryAttempt()
	RecordRetrySuccess()
	RecordRetryExhausted() // All retries failed

	// Get current snapshot
	GetSnapshot() MetricsSnapshot

	// Lifecycle
	Start(ctx context.Context, poolName string)
	Stop(ctx context.Context)
}

// MetricsSnapshot represents a point-in-time view of pool metrics
type MetricsSnapshot struct {
	// Worker info
	WorkersStarted int64 `json:"workers_started"`
	WorkersStopped int64 `json:"workers_stopped"`
	WorkersActive  int64 `json:"workers_active"`
	WorkerPanics   int64 `json:"worker_panics"`

	// Task flow
	TasksCheckedOut int64 `json:"tasks_checked_out"`
	TasksCompleted  int64 `json:"tasks_completed"`
	TasksFailed     int64 `json:"tasks_failed"`
	TasksInProgress int64 `json:"tasks_in_progress"`
	CheckoutErrors  int64 `json:"checkout_errors"`

	// Retry info
	RetryAttempts    int64   `json:"retry_attempts"`
	RetrySuccesses   int64   `json:"retry_successes"`
	RetriesExhausted int64   `json:"retries_exhausted"`
	RetryRate        float64 `json:"retry_rate"` // Percentage of tasks that needed retry

	// Performance
	TotalDuration   time.Duration `json:"total_duration_ms"`
	AverageDuration time.Duration `json:"average_duration_ms"`
	MinDuration     time.Duration `json:"min_duration_ms"`
	MaxDuration     time.Duration `json:"max_duration_ms"`

	// Throughput
	Throughput float64 `json:"throughput_per_sec"`
	ErrorRate  float64 `json:"error_rate"`

	// Timing
	CollectedAt    time.Time     `json:"collected_at"`
	UptimeDuration time.Duration `json:"uptime_seconds"`
}

// ================================================================================
// NoOpMetrics - Default no-op implementation
// ================================================================================

type NoOpMetrics struct{}

func NewNoOpMetrics() WorkerPoolMetrics {
	return &NoOpMetrics{}
}

func (n *NoOpMetrics) RecordWorkerStarted()                       {}
func (n *NoOpMetrics) RecordWorkerStopped()                       {}
func (n *NoOpMetrics) RecordWorkerPanic()                         {}
func (n *NoOpMetrics) RecordTaskCheckedOut()                      {}
func (n *NoOpMetrics) RecordTaskCompleted(duration time.Duration) {}
func (n *NoOpMetrics) RecordTaskFailed(duration time.Duration)    {}
func (n *NoOpMetrics) RecordCheckoutError()                       {}
func (n *NoOpMetrics) RecordRetryAttempt()                        {}
func (n *NoOpMetrics) RecordRetrySuccess()                        {}
func (n *NoOpMetrics) RecordRetryExhausted()                      {}
func (n *NoOpMetrics) GetSnapshot() MetricsSnapshot               { return MetricsSnapshot{} }
func (n *NoOpMetrics) Start(ctx context.Context, poolName string) {}
func (n *NoOpMetrics) Stop(ctx context.Context)                   {}

// ================================================================================
// InMemoryMetrics - Tracks metrics in memory
// ================================================================================

type InMemoryMetrics struct {
	poolName  string
	startTime time.Time

	// Atomics for thread-safe counting
	workersStarted atomic.Int64
	workersStopped atomic.Int64
	workerPanics   atomic.Int64

	tasksCheckedOut atomic.Int64
	tasksCompleted  atomic.Int64
	tasksFailed     atomic.Int64
	checkoutErrors  atomic.Int64

	retryAttempts    atomic.Int64
	retrySuccesses   atomic.Int64
	retriesExhausted atomic.Int64

	totalDurationNs atomic.Int64

	// Protected by mutex for min/max tracking
	mu          sync.RWMutex
	minDuration time.Duration
	maxDuration time.Duration
}

func NewInMemoryMetrics() WorkerPoolMetrics {
	return &InMemoryMetrics{
		minDuration: time.Duration(1<<63 - 1), // MaxInt64
	}
}

func (m *InMemoryMetrics) Start(ctx context.Context, poolName string) {
	m.poolName = poolName
	m.startTime = time.Now()
}

func (m *InMemoryMetrics) Stop(ctx context.Context) {
	// Could flush final metrics here if needed
}

func (m *InMemoryMetrics) RecordWorkerStarted() {
	m.workersStarted.Add(1)
}

func (m *InMemoryMetrics) RecordWorkerStopped() {
	m.workersStopped.Add(1)
}

func (m *InMemoryMetrics) RecordWorkerPanic() {
	m.workerPanics.Add(1)
}

func (m *InMemoryMetrics) RecordTaskCheckedOut() {
	m.tasksCheckedOut.Add(1)
}

func (m *InMemoryMetrics) RecordTaskCompleted(duration time.Duration) {
	m.tasksCompleted.Add(1)
	m.totalDurationNs.Add(int64(duration))

	m.mu.Lock()
	if duration < m.minDuration {
		m.minDuration = duration
	}
	if duration > m.maxDuration {
		m.maxDuration = duration
	}
	m.mu.Unlock()
}

func (m *InMemoryMetrics) RecordTaskFailed(duration time.Duration) {
	m.tasksFailed.Add(1)
	m.totalDurationNs.Add(int64(duration))

	m.mu.Lock()
	if duration < m.minDuration {
		m.minDuration = duration
	}
	if duration > m.maxDuration {
		m.maxDuration = duration
	}
	m.mu.Unlock()
}

func (m *InMemoryMetrics) RecordCheckoutError() {
	m.checkoutErrors.Add(1)
}

func (m *InMemoryMetrics) RecordRetryAttempt() {
	m.retryAttempts.Add(1)
}

func (m *InMemoryMetrics) RecordRetrySuccess() {
	m.retrySuccesses.Add(1)
}

func (m *InMemoryMetrics) RecordRetryExhausted() {
	m.retriesExhausted.Add(1)
}

func (m *InMemoryMetrics) GetSnapshot() MetricsSnapshot {
	now := time.Now()
	uptime := now.Sub(m.startTime)

	workersStarted := m.workersStarted.Load()
	workersStopped := m.workersStopped.Load()
	tasksCompleted := m.tasksCompleted.Load()
	tasksFailed := m.tasksFailed.Load()
	tasksCheckedOut := m.tasksCheckedOut.Load()
	totalTasks := tasksCompleted + tasksFailed

	m.mu.RLock()
	minDur := m.minDuration
	maxDur := m.maxDuration
	m.mu.RUnlock()

	// Calculate averages
	var avgDuration time.Duration
	if totalTasks > 0 {
		avgDuration = time.Duration(m.totalDurationNs.Load() / totalTasks)
	}

	// Calculate throughput
	var throughput float64
	if uptime.Seconds() > 0 {
		throughput = float64(totalTasks) / uptime.Seconds()
	}

	// Calculate error rate
	var errorRate float64
	if totalTasks > 0 {
		errorRate = float64(tasksFailed) / float64(totalTasks) * 100
	}

	// Calculate retry rate
	var retryRate float64
	if totalTasks > 0 {
		retryRate = float64(m.retryAttempts.Load()) / float64(totalTasks) * 100
	}

	// Handle case where no tasks have been processed yet
	if minDur == time.Duration(1<<63-1) {
		minDur = 0
	}

	return MetricsSnapshot{
		WorkersStarted: workersStarted,
		WorkersStopped: workersStopped,
		WorkersActive:  workersStarted - workersStopped,
		WorkerPanics:   m.workerPanics.Load(),

		TasksCheckedOut: tasksCheckedOut,
		TasksCompleted:  tasksCompleted,
		TasksFailed:     tasksFailed,
		TasksInProgress: tasksCheckedOut - totalTasks,
		CheckoutErrors:  m.checkoutErrors.Load(),

		RetryAttempts:    m.retryAttempts.Load(),
		RetrySuccesses:   m.retrySuccesses.Load(),
		RetriesExhausted: m.retriesExhausted.Load(),
		RetryRate:        retryRate,

		TotalDuration:   time.Duration(m.totalDurationNs.Load()),
		AverageDuration: avgDuration,
		MinDuration:     minDur,
		MaxDuration:     maxDur,

		Throughput: throughput,
		ErrorRate:  errorRate,

		CollectedAt:    now,
		UptimeDuration: uptime,
	}
}

// ================================================================================
// LoggerMetrics - Logs metrics using slog
// ================================================================================

type LoggerMetrics struct {
	*InMemoryMetrics
	interval     time.Duration
	taskInterval int64
	logOnStop    bool
	level        slog.Level
	logger       *slog.Logger

	ticker *time.Ticker
	done   chan bool
}

// MetricsOption is a functional option for configuring metrics
type MetricsOption func(*metricsOptions)

type metricsOptions struct {
	interval     time.Duration
	taskInterval int64
	logOnStop    bool
	level        slog.Level
	logger       *slog.Logger
}

// WithInterval sets how often to log metrics
func WithMetricsInterval(interval time.Duration) MetricsOption {
	return func(o *metricsOptions) {
		o.interval = interval
	}
}

// WithTaskInterval sets logging every N tasks
func WithMetricsTaskInterval(count int64) MetricsOption {
	return func(o *metricsOptions) {
		o.taskInterval = count
	}
}

// WithLogOnStop enables/disables logging on stop
func WithMetricsLogOnStop(enabled bool) MetricsOption {
	return func(o *metricsOptions) {
		o.logOnStop = enabled
	}
}

// WithLogLevel sets the log level for metrics
func WithMetricsLogLevel(level slog.Level) MetricsOption {
	return func(o *metricsOptions) {
		o.level = level
	}
}

// WithSlogLogger sets a custom slog logger
func WithMetricsLogger(logger *slog.Logger) MetricsOption {
	return func(o *metricsOptions) {
		o.logger = logger
	}
}

// NewLoggerMetrics creates metrics with structured logging
func NewLoggerMetrics(opts ...MetricsOption) WorkerPoolMetrics {
	// Start with defaults
	options := &metricsOptions{
		interval:     30 * time.Second,
		taskInterval: 100,
		logOnStop:    true,
		level:        slog.LevelInfo,
		logger:       nil,
	}

	// Apply options
	for _, opt := range opts {
		opt(options)
	}

	// Set default logger if not provided
	if options.logger == nil {
		options.logger = slog.Default()
	}

	return &LoggerMetrics{
		InMemoryMetrics: &InMemoryMetrics{
			minDuration: time.Duration(1<<63 - 1),
		},
		interval:     options.interval,
		taskInterval: options.taskInterval,
		logOnStop:    options.logOnStop,
		level:        options.level,
		logger:       options.logger,
		done:         make(chan bool),
	}
}

// NewStdoutMetrics creates metrics that log to stdout with text format
func NewStdoutMetrics(opts ...MetricsOption) WorkerPoolMetrics {
	// Create default options for stdout
	defaultOpts := []MetricsOption{
		WithMetricsInterval(30 * time.Second),
		WithMetricsTaskInterval(0),
		WithMetricsLogOnStop(true),
		WithMetricsLogLevel(slog.LevelInfo),
	}

	// User options override defaults
	allOpts := append(defaultOpts, opts...)

	// Add stdout logger as final option (unless user provided one)
	hasLogger := false
	for _, opt := range opts {
		// Check if WithSlogLogger was used (a bit hacky but works)
		o := &metricsOptions{}
		opt(o)
		if o.logger != nil {
			hasLogger = true
			break
		}
	}

	if !hasLogger {
		handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		stdoutLogger := slog.New(handler)
		allOpts = append(allOpts, WithMetricsLogger(stdoutLogger))
	}

	return NewLoggerMetrics(allOpts...)
}

func (l *LoggerMetrics) Start(ctx context.Context, poolName string) {
	l.InMemoryMetrics.Start(ctx, poolName)

	if l.interval > 0 {
		l.ticker = time.NewTicker(l.interval)
		go l.periodicLog(ctx)
	}

	if l.taskInterval > 0 {
		go l.taskIntervalLog(ctx)
	}

	l.logger.LogAttrs(ctx, l.level, "worker_pool_started",
		slog.String("pool", poolName),
		slog.Time("start_time", l.startTime),
		slog.Duration("log_interval", l.interval),
		slog.Int64("task_interval", l.taskInterval),
	)
}

func (l *LoggerMetrics) Stop(ctx context.Context) {
	if l.ticker != nil {
		l.ticker.Stop()
	}
	close(l.done)

	if l.logOnStop {
		snapshot := l.GetSnapshot()
		l.logger.LogAttrs(ctx, l.level, "worker_pool_stopped",
			slog.String("pool", l.poolName),
			slog.Duration("uptime", snapshot.UptimeDuration),
		)
		l.logMetrics(ctx, snapshot, "shutdown")
	}
}

func (l *LoggerMetrics) periodicLog(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-l.done:
			return
		case <-l.ticker.C:
			snapshot := l.GetSnapshot()
			l.logMetrics(ctx, snapshot, "periodic")
		}
	}
}

func (l *LoggerMetrics) taskIntervalLog(ctx context.Context) {
	lastCount := int64(0)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.done:
			return
		case <-ticker.C:
			total := l.tasksCompleted.Load() + l.tasksFailed.Load()
			if total-lastCount >= l.taskInterval {
				snapshot := l.GetSnapshot()
				l.logMetrics(ctx, snapshot, "task_interval")
				lastCount = total
			}
		}
	}
}

func (l *LoggerMetrics) logMetrics(ctx context.Context, snapshot MetricsSnapshot, trigger string) {
	attrs := []slog.Attr{
		slog.String("pool", l.poolName),
		slog.String("trigger", trigger),
		slog.Time("collected_at", snapshot.CollectedAt),
		slog.Duration("uptime", snapshot.UptimeDuration.Round(time.Second)),

		// Workers group
		slog.Group("workers",
			slog.Int64("active", snapshot.WorkersActive),
			slog.Int64("started", snapshot.WorkersStarted),
			slog.Int64("stopped", snapshot.WorkersStopped),
			slog.Int64("panics", snapshot.WorkerPanics),
		),

		// Tasks group
		slog.Group("tasks",
			slog.Int64("completed", snapshot.TasksCompleted),
			slog.Int64("failed", snapshot.TasksFailed),
			slog.Int64("in_progress", snapshot.TasksInProgress),
			slog.Int64("checkout_errors", snapshot.CheckoutErrors),
		),

		// Performance group
		slog.Group("performance",
			slog.Duration("avg_duration", snapshot.AverageDuration),
			slog.Duration("min_duration", snapshot.MinDuration),
			slog.Duration("max_duration", snapshot.MaxDuration),
			slog.Float64("throughput_per_sec", snapshot.Throughput),
			slog.Float64("error_rate_pct", snapshot.ErrorRate),
		),
	}

	// Add retry metrics if present
	if snapshot.RetryAttempts > 0 {
		attrs = append(attrs, slog.Group("retries",
			slog.Int64("attempts", snapshot.RetryAttempts),
			slog.Int64("successes", snapshot.RetrySuccesses),
			slog.Int64("exhausted", snapshot.RetriesExhausted),
			slog.Float64("retry_rate_pct", snapshot.RetryRate),
		))
	}

	l.logger.LogAttrs(ctx, l.level, "worker_pool_metrics", attrs...)
}
