// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// Type alias provides zero-cost access to generated filter type.
// To extend with custom filters, change from alias to struct embedding:
//
// From:  type UserSessionFilter = GeneratedUserSessionFilter
// To:    type UserSessionFilter struct {
//            GeneratedUserSessionFilter
//            CustomFilter string `json:"custom_filter,omitempty"`
//        }

package usersessionsrepo

// ========================================
// FILTER TYPE ALIAS
// ========================================

// UserSessionFilter holds the available fields a query can be filtered on.
// This is a type alias to GeneratedUserSessionFilter for zero-cost abstraction.
// Change to struct embedding if you need to add custom filter fields.
type UserSessionFilter = GeneratedUserSessionFilter
