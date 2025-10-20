// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// The bridge struct embeds GeneratedBridge to inherit all HTTP handler methods.
// You can override any generated method by defining it here on the bridge type.
// You can also add custom HTTP handler methods here.

package usersrepobridge

import "github.com/jrazmi/envoker/core/repositories/usersrepo"

// ========================================
// BRIDGE STRUCT WITH EMBEDDING
// ========================================

// bridge provides HTTP handlers for User operations.
// It embeds GeneratedBridge to inherit all generated HTTP handler methods.
// Override any method by defining it on this struct.
type bridge struct {
	GeneratedBridge
}

// newBridge creates a new User bridge
func newBridge(userRepository *usersrepo.Repository) *bridge {
	return &bridge{
		GeneratedBridge: GeneratedBridge{
			userRepository: userRepository,
		},
	}
}
