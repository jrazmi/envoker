package workers

import (
	"context"
	"errors"
	"sync"
)

// buildMiddlewareChain creates the middleware chain
func (wp *WorkerPool[T]) buildMiddlewareChain() {
	// Start with the base work function
	wp.workFunc = wp.work

	// Apply middlewares in reverse order (so first added = outermost)
	for i := len(wp.middlewares) - 1; i >= 0; i-- {
		wp.workFunc = wp.middlewares[i](wp.workFunc)
	}
}

func ConsecutiveErrorShutdown(count int) Middleware {
	errorCounts := make(map[string]int)
	var mu sync.Mutex

	return func(next WorkFunc) WorkFunc {
		return func(ctx context.Context, workerID string) error {
			err := next(ctx, workerID)
			mu.Lock()
			if err != nil && !errors.Is(err, ErrNoWorkAvailable) {
				errorCounts[workerID]++
				if errorCounts[workerID] > count {
					mu.Unlock()
					return ErrWorkerShutdown // Stop this worker
				}
			} else if err == nil {
				errorCounts[workerID] = 0
			}
			mu.Unlock()

			return err
		}
	}
}
