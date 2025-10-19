package bridgegen

// ModelCustomTemplate is the template for model.go (generated only if doesn't exist)
// This file provides type aliases for the bridge models, which can be changed to struct embedding if needed
const ModelCustomTemplate = `// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// Type aliases provide zero-cost access to generated types.
// To extend a type, change from alias to struct embedding:
//
// From:  type {{.EntityName}} = Generated{{.EntityName}}
// To:    type {{.EntityName}} struct {
//            Generated{{.EntityName}}
//            CustomField string ` + "`" + `json:"custom_field"` + "`" + `
//        }

package {{.PackageName}}

// ========================================
// BRIDGE MODEL TYPE ALIASES
// ========================================

// {{.EntityName}} is the bridge model for {{.EntityNameLower}}.
// This is a type alias to Generated{{.EntityName}} for zero-cost abstraction.
// Change to struct embedding if you need to add custom fields.
type {{.EntityName}} = Generated{{.EntityName}}

// Create{{.EntityName}}Input contains fields for creating a new {{.EntityNameLower}}.
// Change to struct embedding if you need to add custom validation or fields.
type Create{{.EntityName}}Input = GeneratedCreate{{.EntityName}}Input

// Update{{.EntityName}}Input contains fields for updating an existing {{.EntityNameLower}}.
// All fields are optional to support partial updates.
// Change to struct embedding if you need to add custom fields or validation.
type Update{{.EntityName}}Input = GeneratedUpdate{{.EntityName}}Input
`
