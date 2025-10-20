// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// Type aliases provide zero-cost access to generated types.
// To extend a type, change from alias to struct embedding:
//
// From:  type Task = GeneratedTask
// To:    type Task struct {
//            GeneratedTask
//            CustomField string `json:"custom_field"`
//        }

package tasksrepobridge

// ========================================
// BRIDGE MODEL TYPE ALIASES
// ========================================

// Task is the bridge model for task.
// This is a type alias to GeneratedTask for zero-cost abstraction.
// Change to struct embedding if you need to add custom fields.
type Task = GeneratedTask

// CreateTaskInput contains fields for creating a new task.
// Change to struct embedding if you need to add custom validation or fields.
type CreateTaskInput = GeneratedCreateTaskInput

// UpdateTaskInput contains fields for updating an existing task.
// All fields are optional to support partial updates.
// Change to struct embedding if you need to add custom fields or validation.
type UpdateTaskInput = GeneratedUpdateTaskInput

// ========================================
// REPOSITORY INTERFACE TYPE ALIAS
// ========================================

// TaskRepository is the repository interface used by the bridge.
// This is a type alias to GeneratedTaskRepository for zero-cost abstraction.
// To extend the interface with additional methods, change to interface embedding:
//
// From:  type TaskRepository = GeneratedTaskRepository
//
//	To:    type TaskRepository interface {
//	           GeneratedTaskRepository
//	           CustomMethod(ctx context.Context, ...) error
//	       }
type TaskRepository = GeneratedTaskRepository
