package repositorygen

// FOPTemplate is the template for fop.go (generated only if doesn't exist)
// This file provides type alias for the filter, which can be changed to struct embedding if needed
const FOPTemplate = `// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// Type alias provides zero-cost access to generated filter type.
// To extend with custom filters, change from alias to struct embedding:
//
// From:  type {{.EntityName}}Filter = Generated{{.EntityName}}Filter
// To:    type {{.EntityName}}Filter struct {
//            Generated{{.EntityName}}Filter
//            CustomFilter string ` + "`" + `json:"custom_filter,omitempty"` + "`" + `
//        }

package {{.PackageName}}

// ========================================
// FILTER TYPE ALIAS
// ========================================

// {{.EntityName}}Filter holds the available fields a query can be filtered on.
// This is a type alias to Generated{{.EntityName}}Filter for zero-cost abstraction.
// Change to struct embedding if you need to add custom filter fields.
type {{.EntityName}}Filter = Generated{{.EntityName}}Filter
`
