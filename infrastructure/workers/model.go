package workers

import "context"

// Task interface - any task must have an ID
type Task interface {
	GetID() string
}

// Processor handles the business logic for processing tasks
type Processor[T Task] interface {
	// Checkout gets the next available task (must be atomic for concurrent workers)
	Checkout(ctx context.Context, workerID string) (T, error)

	// Process executes the task and returns a result
	Process(ctx context.Context, task T) (T, error)

	// Complete is called when a task completes successfully
	Complete(ctx context.Context, task T, processingTimeMS int) error // Now takes typed result

	// Fail is called when a task fails
	Fail(ctx context.Context, task T, err error) error
}

// WorkFunc is the signature for the work function
type WorkFunc func(ctx context.Context, workerID string) error

// Middleware wraps a WorkFunc with additional behavior
type Middleware func(WorkFunc) WorkFunc

// PreProcessHook runs before Process
type PreProcessHook[T Task] func(ctx context.Context, task T) error

// PostProcessHook runs after Process (gets the result or error)
type PostProcessHook[T Task] func(ctx context.Context, task T, err error) error
