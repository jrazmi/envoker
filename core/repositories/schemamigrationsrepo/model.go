// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// Type aliases provide zero-cost access to generated types.
// To extend a type, change from alias to struct embedding:
//
// From:  type SchemaMigration = GeneratedSchemaMigration
// To:    type SchemaMigration struct {
//            GeneratedSchemaMigration
//            CustomField string `json:"custom_field"`
//        }

package schemamigrationsrepo

// ========================================
// MODEL TYPE ALIASES
// ========================================

// SchemaMigration is the main entity type.
// This is a type alias to GeneratedSchemaMigration for zero-cost abstraction.
// Change to struct embedding if you need to add custom fields.
type SchemaMigration = GeneratedSchemaMigration

// CreateSchemaMigration contains fields for creating a new schemaMigration.
// Change to struct embedding if you need to add custom validation or fields.
type CreateSchemaMigration = GeneratedCreateSchemaMigration

// UpdateSchemaMigration contains fields for updating an existing schemaMigration.
// All fields are optional (pointers) to support partial updates.
// Change to struct embedding if you need to add custom fields or validation.
type UpdateSchemaMigration = GeneratedUpdateSchemaMigration
