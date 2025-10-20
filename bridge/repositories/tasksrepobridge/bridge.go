// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// The bridge struct embeds GeneratedBridge to inherit all HTTP handler methods.
// You can override any generated method by defining it here on the bridge type.
// You can also add custom HTTP handler methods here.

package tasksrepobridge

import "github.com/jrazmi/envoker/core/repositories/tasksrepo"

// ========================================
// BRIDGE STRUCT WITH EMBEDDING
// ========================================

// bridge provides HTTP handlers for Task operations.
// It embeds GeneratedBridge to inherit all generated HTTP handler methods.
// Override any method by defining it on this struct.
type bridge struct {
	GeneratedBridge
}

// newBridge creates a new Task bridge
func newBridge(taskRepository *tasksrepo.Repository) *bridge {
	return &bridge{
		GeneratedBridge: GeneratedBridge{
			taskRepository: taskRepository,
		},
	}
}
