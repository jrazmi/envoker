package bridgegen

// BridgeInitTemplate is the template for bridge.go (generated only if doesn't exist)
const BridgeInitTemplate = `// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// The bridge struct embeds GeneratedBridge to inherit all HTTP handler methods.
// You can override any generated method by defining it here on the bridge type.
// You can also add custom HTTP handler methods here.

package {{.BridgePackage}}

import "{{.ModulePath}}/core/repositories/{{.RepoPackage}}"

// ========================================
// BRIDGE STRUCT WITH EMBEDDING
// ========================================

// bridge provides HTTP handlers for {{.Entity}} operations.
// It embeds GeneratedBridge to inherit all generated HTTP handler methods.
// Override any method by defining it on this struct.
type bridge struct {
	GeneratedBridge
}

// newBridge creates a new {{.Entity}} bridge
func newBridge({{.EntityNameLower}}Repository *{{.RepoPackage}}.Repository) *bridge {
	return &bridge{
		GeneratedBridge: GeneratedBridge{
			{{.EntityNameLower}}Repository: {{.EntityNameLower}}Repository,
		},
	}
}
`
