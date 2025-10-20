package fopbridge

import "encoding/json"

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

func NewCodeResponse(code, message string) CodeResponse {
	return CodeResponse{Code: code, Message: message}
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

func (r RecordResponse[T]) Encode() ([]byte, string, error) {
	data, err := json.Marshal(r)
	return data, "application/json", err
}
