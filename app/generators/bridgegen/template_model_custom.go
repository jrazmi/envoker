package bridgegen

// ModelCustomTemplate is the template for model.go (generated only if doesn't exist)
// This file is reserved for custom bridge-specific types and extensions if needed.
const ModelCustomTemplate = `// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// The bridge uses repository types directly ({{.RepoPackage}}.{{.EntityName}}, etc.)
// for maximum simplicity and to avoid duplication.
//
// Use this file to define:
// - Custom request/response wrappers specific to this bridge
// - Bridge-specific validation logic
// - Any custom types needed for HTTP handling
//
// Example:
//   type Custom{{.EntityName}}Response struct {
//       {{.EntityName}} {{.RepoPackage}}.{{.EntityName}} ` + "`" + `json:"{{.EntityNameLower}}"` + "`" + `
//       ComputedField string ` + "`" + `json:"computed_field"` + "`" + `
//   }

package {{.PackageName}}

// Add your custom bridge types here
`
