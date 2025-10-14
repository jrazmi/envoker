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
// Unified Pagination Response
// ============================================================================

// PaginatedResponse is a unified response type for all cursor types
type PaginatedResponse[T any, C comparable] struct {
	Records  []T         `json:"records"`
	PageInfo PageInfo[C] `json:"pageInfo"`
}

// PageInfo is a generic page info structure that works with any cursor type
type PageInfo[C comparable] struct {
	HasPrev        bool `json:"hasPrev,omitempty"`
	HasNext        bool `json:"hasNext,omitempty"`
	Limit          int  `json:"limit,omitempty"`
	PreviousCursor *C   `json:"previousCursor,omitempty"`
	NextCursor     *C   `json:"nextCursor,omitempty"`
	PageTotal      int  `json:"pageTotal,omitempty"`
}

// Encode implements the encoder interface for the paginated response
func (p PaginatedResponse[T, C]) Encode() ([]byte, string, error) {
	data, err := json.Marshal(p)
	return data, "application/json", err
}

// ============================================================================
// Unified Constructor Functions
// ============================================================================

// NewPaginatedResponse creates a paginated response for any cursor type
func NewPaginatedResponse[T any, C comparable](records []T, pageInfo PageInfo[C]) PaginatedResponse[T, C] {
	return PaginatedResponse[T, C]{
		Records:  records,
		PageInfo: pageInfo,
	}
}

// NewPaginatedResponseFromPage creates a paginated response from fop page types
// This is the main function you'll use - it handles all cursor types automatically
func NewPaginatedResponseFromPage[T any](records []T, page any) interface{} {
	switch p := page.(type) {
	case fop.PageStringCursor:
		return NewPaginatedResponseStringCursor(records, p)
	case fop.PageInfoStringCursor:
		return NewPaginatedResponseFromStringCursorInfo(records, p)
	case fop.PageInt64Cursor:
		return NewPaginatedResponseInt64Cursor(records, p)
	case fop.PageIntCursor:
		return NewPaginatedResponseIntCursor(records, p)
	default:
		// Fallback to non-paginated response
		return NonPaginatedRecords[T]{Records: records}
	}
}

// ============================================================================
// Specific Cursor Type Constructors
// ============================================================================

// String cursor support
func NewPaginatedResponseStringCursor[T any](records []T, page fop.PageStringCursor) PaginatedResponse[T, string] {
	pageInfo := PageInfo[string]{
		HasPrev:   page.Cursor != "",
		HasNext:   len(records) == page.Limit,
		Limit:     page.Limit,
		PageTotal: len(records),
	}

	// Set cursors based on records
	if len(records) > 0 && len(records) == page.Limit {
		// Assuming records have an ID field or similar for next cursor
		// You might need to adjust this based on your record structure
		nextCursor := extractStringCursor(records[len(records)-1])
		pageInfo.NextCursor = &nextCursor
	}

	if page.Cursor != "" {
		// Calculate previous cursor if needed
		prevCursor := page.Cursor // Simplified - you might need better logic
		pageInfo.PreviousCursor = &prevCursor
	}

	return PaginatedResponse[T, string]{
		Records:  records,
		PageInfo: pageInfo,
	}
}

// String cursor from PageInfo
func NewPaginatedResponseFromStringCursorInfo[T any](records []T, pageInfo fop.PageInfoStringCursor) PaginatedResponse[T, string] {
	genericPageInfo := PageInfo[string]{
		HasPrev:   pageInfo.HasPrev,
		Limit:     pageInfo.Limit,
		PageTotal: len(records),
	}

	if pageInfo.PreviousCursor != "" {
		genericPageInfo.PreviousCursor = &pageInfo.PreviousCursor
	}
	if pageInfo.NextCursor != "" {
		genericPageInfo.NextCursor = &pageInfo.NextCursor
	}

	return PaginatedResponse[T, string]{
		Records:  records,
		PageInfo: genericPageInfo,
	}
}

// Int64 cursor support
func NewPaginatedResponseInt64Cursor[T any](records []T, page fop.PageInt64Cursor) PaginatedResponse[T, int64] {
	var prevCursor, nextCursor *int64

	if page.Cursor > int64(page.Limit) {
		prev := page.Cursor - int64(page.Limit)
		prevCursor = &prev
	}

	if len(records) == page.Limit {
		next := page.Cursor + int64(page.Limit)
		nextCursor = &next
	}

	pageInfo := PageInfo[int64]{
		HasPrev:        prevCursor != nil,
		HasNext:        nextCursor != nil,
		Limit:          page.Limit,
		PreviousCursor: prevCursor,
		NextCursor:     nextCursor,
		PageTotal:      len(records),
	}

	return PaginatedResponse[T, int64]{
		Records:  records,
		PageInfo: pageInfo,
	}
}

// Int cursor support
func NewPaginatedResponseIntCursor[T any](records []T, page fop.PageIntCursor) PaginatedResponse[T, int] {
	var prevCursor, nextCursor *int

	if page.Cursor > page.Limit {
		prev := page.Cursor - page.Limit
		prevCursor = &prev
	}

	if len(records) == page.Limit {
		next := page.Cursor + page.Limit
		nextCursor = &next
	}

	pageInfo := PageInfo[int]{
		HasPrev:        prevCursor != nil,
		HasNext:        nextCursor != nil,
		Limit:          page.Limit,
		PreviousCursor: prevCursor,
		NextCursor:     nextCursor,
		PageTotal:      len(records),
	}

	return PaginatedResponse[T, int]{
		Records:  records,
		PageInfo: pageInfo,
	}
}

// ============================================================================
// Non-Paginated Response
// ============================================================================

type NonPaginatedRecords[T any] struct {
	Records []T `json:"records"`
}

func NewNonPaginatedRecords[T any](records []T) NonPaginatedRecords[T] {
	return NonPaginatedRecords[T]{Records: records}
}

func (n NonPaginatedRecords[T]) Encode() ([]byte, string, error) {
	data, err := json.Marshal(n)
	return data, "application/json", err
}

// ============================================================================
// Helper Functions
// ============================================================================

// extractStringCursor extracts a cursor value from a record
// This is a placeholder - you'll need to implement based on your record structure
func extractStringCursor[T any](record T) string {
	// This would need to be implemented based on your specific record types
	// For example, if your records have an ID field:
	// if r, ok := any(record).(interface{ GetID() string }); ok {
	//     return r.GetID()
	// }
	return ""
}

// ============================================================================
// Convenience Functions for Common Use Cases
// ============================================================================

// Simple constructors that match your existing patterns
func NewPaginatedResultStringCursor[T any](records []T, pageInfo fop.PageInfoStringCursor) PaginatedResponse[T, string] {
	return NewPaginatedResponseFromStringCursorInfo(records, pageInfo)
}

func NewPaginatedResultInt64Offset[T any](records []T, page fop.PageInt64Cursor) PaginatedResponse[T, int64] {
	return NewPaginatedResponseInt64Cursor(records, page)
}
