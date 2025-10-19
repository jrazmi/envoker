package repositorygen

// ModelTemplate is the template for model.go (generated only if doesn't exist)
// This file provides type aliases for the models, which can be changed to struct embedding if needed
const ModelTemplate = `// This file is only generated if it doesn't already exist.
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
// MODEL TYPE ALIASES
// ========================================

// {{.EntityName}} is the main entity type.
// This is a type alias to Generated{{.EntityName}} for zero-cost abstraction.
// Change to struct embedding if you need to add custom fields.
type {{.EntityName}} = Generated{{.EntityName}}

// {{.CreateStructName}} contains fields for creating a new {{.EntityNameLower}}.
// Change to struct embedding if you need to add custom validation or fields.
type {{.CreateStructName}} = Generated{{.CreateStructName}}

// {{.UpdateStructName}} contains fields for updating an existing {{.EntityNameLower}}.
// All fields are optional (pointers) to support partial updates.
// Change to struct embedding if you need to add custom fields or validation.
type {{.UpdateStructName}} = Generated{{.UpdateStructName}}
`
