// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// The bridge struct embeds GeneratedBridge to inherit all HTTP handler methods.
// You can override any generated method by defining it here on the bridge type.
// You can also add custom HTTP handler methods here.

package usersessionsrepobridge

import "github.com/jrazmi/envoker/core/repositories/usersessionsrepo"

// ========================================
// BRIDGE STRUCT WITH EMBEDDING
// ========================================

// bridge provides HTTP handlers for UserSession operations.
// It embeds GeneratedBridge to inherit all generated HTTP handler methods.
// Override any method by defining it on this struct.
type bridge struct {
	GeneratedBridge
}

// newBridge creates a new UserSession bridge
func newBridge(userSessionRepository *usersessionsrepo.Repository) *bridge {
	return &bridge{
		GeneratedBridge: GeneratedBridge{
			userSessionRepository: userSessionRepository,
		},
	}
}
