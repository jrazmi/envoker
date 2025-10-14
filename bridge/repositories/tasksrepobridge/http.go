// bridge/repositories/tasksrepobridge/routes.go
package tasksrepobridge

import (
	"context"
	"net/http"

	"github.com/jrazmi/envoker/bridge/scaffolding/errs"
	"github.com/jrazmi/envoker/bridge/scaffolding/fopbridge"
	"github.com/jrazmi/envoker/core/repositories/tasksrepo"
	"github.com/jrazmi/envoker/core/scaffolding/fop"
	"github.com/jrazmi/envoker/infrastructure/web"
	"github.com/jrazmi/envoker/sdk/logger"
)

type Config struct {
	Log        *logger.Logger
	Repository *tasksrepo.Repository
	Middleware []web.Middleware
}

func AddHttpRoutes(group *web.RouteGroup, cfg Config) {
	bridge := newBridge(cfg.Repository)

	// Standard CRUD routes (relative to the group)
	group.GET("/tasks", bridge.httpList)
	group.GET("/tasks/{task_id}", bridge.httpGetByID)
	group.POST("/tasks", bridge.httpCreate)
	group.PUT("/tasks/{task_id}", bridge.httpUpdate)

	// Special Python worker routes
	group.GET("/tasks/checkout/{task_type}", bridge.httpCheckoutTask)
	group.POST("/tasks/complete", bridge.httpCompleteTask)
	group.POST("/tasks/fail", bridge.httpFailTask)
}

// list handles GET requests for listing tasks with pagination and filtering
func (b *bridge) httpList(ctx context.Context, r *http.Request) web.Encoder {
	qp := parseQueryParams(r)

	page, err := fop.ParsePageStringCursor(qp.Limit, qp.Cursor)
	if err != nil {
		return errs.NewFieldErrors("page", err)
	}

	filter, err := parseFilter(qp)
	if err != nil {
		return errs.NewFieldErrors("filter", err)
	}

	orderBy := parseOrderBy(qp.Order)

	records, pageInfo, err := b.tasksRepository.List(ctx, filter, orderBy, page)
	if err != nil {
		return errs.Newf(errs.Internal, "list tasks: %s", err)
	}

	// Your existing paginated response works perfectly!
	return fopbridge.NewPaginatedResultStringCursor(MarshalListToBridge(records), pageInfo)
}

// getByID handles GET requests for retrieving a specific task by ID
func (b *bridge) httpGetByID(ctx context.Context, r *http.Request) web.Encoder {
	qpath, err := parsePath(r)
	if err != nil {
		return errs.Newf(errs.InvalidArgument, "invalid path arguments: %s", err)
	}

	if qpath.TaskID == "" {
		return errs.Newf(errs.InvalidArgument, "task_id is required")
	}

	qp := parseQueryParams(r)
	filter, err := parseFilter(qp)
	if err != nil {
		return errs.NewFieldErrors("filter", err)
	}

	task, err := b.tasksRepository.Get(ctx, qpath.TaskID, filter)
	if err != nil {
		if err == tasksrepo.ErrNotFound {
			return errs.Newf(errs.NotFound, "task not found: %s", qpath.TaskID)
		}
		return errs.Newf(errs.Internal, "get task: %s", err)
	}

	return MarshalToBridge(task)
}

// checkoutTask handles GET requests for checking out the next available task
func (b *bridge) httpCheckoutTask(ctx context.Context, r *http.Request) web.Encoder {
	qpath, err := parsePath(r)
	if err != nil {
		return errs.Newf(errs.InvalidArgument, "invalid path arguments: %s", err)
	}

	if qpath.TaskType == "" {
		return errs.Newf(errs.InvalidArgument, "task_type is required")
	}

	// Checkout task using your existing logic
	task, err := b.tasksRepository.CheckoutForProcessing(ctx, qpath.TaskType)
	if err != nil {
		if err == tasksrepo.ErrNoWorkAvailable {
			// Return 204 No Content when no work available - Python will poll again
			return fopbridge.CodeResponse{
				Code:    errs.NoContent.String(),
				Message: "No work available",
			}
		}
		return errs.Newf(errs.Internal, "checkout task: %s", err)
	}

	return MarshalToBridge(task)
}

// create handles POST requests for creating a new task
func (b *bridge) httpCreate(ctx context.Context, r *http.Request) web.Encoder {
	var input CreateTaskInput
	if err := web.Decode(r, &input); err != nil {
		return errs.Newf(errs.InvalidArgument, "decode: %s", err)
	}

	createInput := MarshalCreateToRepository(input)

	task, err := b.tasksRepository.Create(ctx, createInput)
	if err != nil {
		return errs.Newf(errs.Internal, "create task: %s", err)
	}

	return MarshalToBridge(task)
}

// update handles PUT/PATCH requests for updating an existing task
func (b *bridge) httpUpdate(ctx context.Context, r *http.Request) web.Encoder {
	qpath, err := parsePath(r)
	if err != nil {
		return errs.Newf(errs.InvalidArgument, "invalid path arguments: %s", err)
	}

	if qpath.TaskID == "" {
		return errs.Newf(errs.InvalidArgument, "task_id is required")
	}

	var input UpdateTaskInput
	if err := web.Decode(r, &input); err != nil {
		return errs.Newf(errs.InvalidArgument, "decode: %s", err)
	}

	updateInput := MarshalUpdateToRepository(input)

	err = b.tasksRepository.Update(ctx, qpath.TaskID, updateInput)
	if err != nil {
		if err == tasksrepo.ErrNotFound {
			return errs.Newf(errs.NotFound, "task not found: %s", qpath.TaskID)
		}
		return errs.Newf(errs.Internal, "update task: %s", err)
	}

	return fopbridge.CodeResponse{
		Code:    errs.OK.String(),
		Message: "Task updated successfully",
	}
}

// delete handles DELETE requests for removing a task
func (b *bridge) httpDelete(ctx context.Context, r *http.Request) web.Encoder {
	qpath, err := parsePath(r)
	if err != nil {
		return errs.Newf(errs.InvalidArgument, "invalid path arguments: %s", err)
	}

	if qpath.TaskID == "" {
		return errs.Newf(errs.InvalidArgument, "task_id is required")
	}

	err = b.tasksRepository.Delete(ctx, qpath.TaskID)
	if err != nil {
		if err == tasksrepo.ErrNotFound {
			return errs.Newf(errs.NotFound, "task not found: %s", qpath.TaskID)
		}
		return errs.Newf(errs.Internal, "delete task: %s", err)
	}

	return fopbridge.CodeResponse{
		Code:    errs.OK.String(),
		Message: "Task deleted successfully",
	}
}

// completeTask handles POST requests for marking a task as completed
func (b *bridge) httpCompleteTask(ctx context.Context, r *http.Request) web.Encoder {
	var input CompleteTaskInput
	if err := web.Decode(r, &input); err != nil {
		return errs.Newf(errs.InvalidArgument, "decode: %s", err)
	}

	// Mark task as completed using your existing logic
	if err := b.tasksRepository.MarkCompleted(ctx, input.TaskID, input.ProcessingTimeMs); err != nil {
		return errs.Newf(errs.Internal, "mark task completed: %s", err)
	}

	// DONT DO THIS IT OVERWRITES THE METADATA
	// // Optionally update metadata with results if provided this
	// if input.Result != nil {
	// 	updateInput := MarshalResultToRepository(input.Result)
	// 	if err := b.tasksRepository.Update(ctx, input.TaskID, updateInput); err != nil {
	// 		// Log error but don't fail the completion
	// 		// The task is already marked complete
	// 	}
	// }

	return fopbridge.CodeResponse{
		Code:    errs.OK.String(),
		Message: "Task completed successfully",
	}
}

// failTask handles POST requests for marking a task as failed
func (b *bridge) httpFailTask(ctx context.Context, r *http.Request) web.Encoder {
	var input FailTaskInput
	if err := web.Decode(r, &input); err != nil {
		return errs.Newf(errs.InvalidArgument, "decode: %s", err)
	}

	// Mark task as failed using your existing logic
	if err := b.tasksRepository.MarkFailed(ctx, input.TaskID, input.ErrorMessage); err != nil {
		return errs.Newf(errs.Internal, "mark task failed: %s", err)
	}

	return fopbridge.CodeResponse{
		Code:    errs.OK.String(),
		Message: "Task marked as failed",
	}
}
