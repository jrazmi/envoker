// This file is only generated if it doesn't already exist.
// Once created, you can customize this file freely - it will NOT be overwritten.
//
// Type alias provides zero-cost access to generated filter type.
// To extend with custom filters, change from alias to struct embedding:
//
// From:  type UserFilter = GeneratedUserFilter
// To:    type UserFilter struct {
//            GeneratedUserFilter
//            CustomFilter string `json:"custom_filter,omitempty"`
//        }

package usersrepo

// ========================================
// FILTER TYPE ALIAS
// ========================================

// UserFilter holds the available fields a query can be filtered on.
// This is a type alias to GeneratedUserFilter for zero-cost abstraction.
// Change to struct embedding if you need to add custom filter fields.
type UserFilter = GeneratedUserFilter
