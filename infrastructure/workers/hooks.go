package workers

import (
	"context"
	"fmt"
)

// Add Pre Process Hooks adds functions in after the processor.Checkout call, but before the processor.Process call.
func (wp *WorkerPool[T]) AddPreProcessHooks(hooks ...PreProcessHook[T]) {
	wp.preProcessHooks = append(wp.preProcessHooks, hooks...)
}

// Add Post Process Hooks adds functions in after the processor.Process call, but before the processor.Complete or processor.Fail call
func (wp *WorkerPool[T]) AddPostProcessHooks(hooks ...PostProcessHook[T]) {
	wp.postProcessHooks = append(wp.postProcessHooks, hooks...)
}

// Pre-process: Send notification
func NotifyStartHook[T Task]() PreProcessHook[T] {
	return func(ctx context.Context, task T) error {
		fmt.Println("=====================")
		fmt.Println("notify start")
		fmt.Println("=====================")
		return nil
	}
}

// post-process: Send notification
func NotifyEndHook[T Task]() PostProcessHook[T] {
	return func(ctx context.Context, task T, err error) error {
		fmt.Println("=====================")
		fmt.Println("notify end")
		fmt.Println("=====================")
		return nil
	}
}
