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

package tasksrepo

// ========================================
// MODEL TYPE ALIASES
// ========================================

// Task is the main entity type.
// This is a type alias to GeneratedTask for zero-cost abstraction.
// Change to struct embedding if you need to add custom fields.
type Task = GeneratedTask

// CreateTask contains fields for creating a new task.
// Change to struct embedding if you need to add custom validation or fields.
type CreateTask = GeneratedCreateTask

// UpdateTask contains fields for updating an existing task.
// All fields are optional (pointers) to support partial updates.
// Change to struct embedding if you need to add custom fields or validation.
type UpdateTask = GeneratedUpdateTask
