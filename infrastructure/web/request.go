// web/request.go - Add this to your web package
package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Param returns the web call parameters from the request.
func Param(r *http.Request, key string) string {
	return r.PathValue(key)
}

// QueryParam returns query parameters from the request.
func QueryParam(r *http.Request, key string) string {
	query := r.URL.Query()
	return query.Get(key)
}

// Decoder represents data that can be decoded.
type Decoder interface {
	Decode(data []byte) error
}

// Validator interface for request validation
type validator interface {
	Validate() error
}

// Decode reads the body of an HTTP request and decodes it into the specified data model.
// If the data model implements the validator interface, the Validate method will be called.
func Decode(r *http.Request, v any) error {
	// Read the request body
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read request body: %w", err)
	}

	// Handle empty body
	if len(data) == 0 {
		return fmt.Errorf("request body is empty")
	}

	// If the value implements Decoder interface, use it
	if decoder, ok := v.(Decoder); ok {
		if err := decoder.Decode(data); err != nil {
			return fmt.Errorf("decode: %w", err)
		}
	} else {
		// Default to JSON decoding
		if err := json.Unmarshal(data, v); err != nil {
			return fmt.Errorf("json decode: %w", err)
		}
	}

	// Validate if the struct implements validator interface
	if validator, ok := v.(validator); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("validation: %w", err)
		}
	}

	return nil
}

// DecodeJSON is a convenience function that explicitly uses JSON decoding
func DecodeJSON(r *http.Request, v interface{}) error {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("unable to read request body: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("request body is empty")
	}

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("json decode: %w", err)
	}

	// Validate if the struct implements validator interface
	if validator, ok := v.(validator); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("validation: %w", err)
		}
	}

	return nil
}

// DecodeForm decodes form data (application/x-www-form-urlencoded)
func DecodeForm(r *http.Request, v interface{}) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("parse form: %w", err)
	}

	// You'd need to implement form-to-struct mapping here
	// This is a placeholder - you might want to use a library like gorilla/schema
	return fmt.Errorf("form decoding not implemented")
}

// DecodeMultipartForm decodes multipart form data
func DecodeMultipartForm(r *http.Request, maxMemory int64, v interface{}) error {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return fmt.Errorf("parse multipart form: %w", err)
	}

	// You'd need to implement multipart-to-struct mapping here
	return fmt.Errorf("multipart form decoding not implemented")
}

// ============================================================================
// Usage Examples in your handlers
// ============================================================================

/*
// Example usage in your handlers:

func (b *bridge) httpCreate(ctx context.Context, r *http.Request) web.Encoder {
	var input CreateTaskInput
	if err := web.Decode(r, &input); err != nil {
		return errs.Newf(errs.InvalidArgument, "decode: %s", err)
	}

	// ... rest of handler
}

// If your input structs implement validation:
type CreateTaskInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	TaskType    string `json:"task_type"`
}

func (c CreateTaskInput) Validate() error {
	if c.Title == "" {
		return fmt.Errorf("title is required")
	}
	if c.TaskType == "" {
		return fmt.Errorf("task_type is required")
	}
	return nil
}

// If your input structs implement custom decoding:
type CustomInput struct {
	Data map[string]interface{}
}

func (c *CustomInput) Decode(data []byte) error {
	// Custom decoding logic
	return json.Unmarshal(data, &c.Data)
}
*/
