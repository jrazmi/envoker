// Package fopbridge provides support for query paging with unified response types.
package fopbridge

import (
	"encoding/json"

	"github.com/jrazmi/envoker/core/scaffolding/fop"
)

// ============================================================================
// Standard Response Types
// ============================================================================

// RecordID is the data model used when returning a create/update ID.
type RecordID struct {
	ID string `json:"id"`
}

// CodeResponse provides a standard response with code and message
type CodeResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c CodeResponse) Encode() ([]byte, string, error) {
	data, err := json.Marshal(c)
	return data, "application/json", err
}

// RecordResponse wraps a single record
type RecordResponse[T any] struct {
	Record T `json:"record"`
}

func NewRecordResponse[T any](record T) RecordResponse[T] {
	return RecordResponse[T]{Record: record}
}

// ============================================================================
// Pagination Response
// ============================================================================

// PaginatedResponse is a unified response type for all cursor types
type PaginatedResponse[T any, C comparable] struct {
	Records    []T           `json:"records"`
	Pagination Pagination[C] `json:"pagination"`
}

// Pagination is a generic pagination structure that works with any cursor type
type Pagination[C comparable] struct {
	HasPrev        bool `json:"has_prev,omitempty"`
	HasNext        bool `json:"has_next,omitempty"`
	Limit          int  `json:"limit,omitempty"`
	PreviousCursor *C   `json:"previous_cursor,omitempty"`
	NextCursor     *C   `json:"next_cursor,omitempty"`
	PageTotal      int  `json:"page_total,omitempty"`
}

// Encode implements the encoder interface for the paginated response
func (p PaginatedResponse[T, C]) Encode() ([]byte, string, error) {
	data, err := json.Marshal(p)
	return data, "application/json", err
}

// ============================================================================
// Constructor Functions
// ============================================================================

// NewPaginatedResult creates a paginated response from repository pagination metadata
// This is the primary function used by all generated HTTP handlers
func NewPaginatedResult[T any](records []T, pagination fop.Pagination) PaginatedResponse[T, string] {
	genericPagination := Pagination[string]{
		HasPrev:   pagination.HasPrev,
		Limit:     pagination.Limit,
		PageTotal: len(records),
	}

	if pagination.PreviousCursor != "" {
		genericPagination.PreviousCursor = &pagination.PreviousCursor
	}
	if pagination.NextCursor != "" {
		genericPagination.NextCursor = &pagination.NextCursor
	}

	return PaginatedResponse[T, string]{
		Records:    records,
		Pagination: genericPagination,
	}
}
