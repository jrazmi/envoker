// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// Type aliases provide zero-cost access to generated types.
// To extend a type, change from alias to struct embedding:
//
// From:  type User = GeneratedUser
// To:    type User struct {
//            GeneratedUser
//            CustomField string `json:"custom_field"`
//        }

package usersrepo

// ========================================
// MODEL TYPE ALIASES
// ========================================

// User is the main entity type.
// This is a type alias to GeneratedUser for zero-cost abstraction.
// Change to struct embedding if you need to add custom fields.
type User = GeneratedUser

// CreateUser contains fields for creating a new user.
// Change to struct embedding if you need to add custom validation or fields.
type CreateUser = GeneratedCreateUser

// UpdateUser contains fields for updating an existing user.
// All fields are optional (pointers) to support partial updates.
// Change to struct embedding if you need to add custom fields or validation.
type UpdateUser = GeneratedUpdateUser
