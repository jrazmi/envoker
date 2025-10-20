// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// Type aliases provide zero-cost access to generated types.
// To extend a type, change from alias to struct embedding:
//
// From:  type UserSession = GeneratedUserSession
// To:    type UserSession struct {
//            GeneratedUserSession
//            CustomField string `json:"custom_field"`
//        }

package usersessionsrepo

// ========================================
// MODEL TYPE ALIASES
// ========================================

// UserSession is the main entity type.
// This is a type alias to GeneratedUserSession for zero-cost abstraction.
// Change to struct embedding if you need to add custom fields.
type UserSession = GeneratedUserSession

// CreateUserSession contains fields for creating a new userSession.
// Change to struct embedding if you need to add custom validation or fields.
type CreateUserSession = GeneratedCreateUserSession

// UpdateUserSession contains fields for updating an existing userSession.
// All fields are optional (pointers) to support partial updates.
// Change to struct embedding if you need to add custom fields or validation.
type UpdateUserSession = GeneratedUpdateUserSession
