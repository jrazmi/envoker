package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// NoResponse tells the Respond function to not respond to the request. In these
// cases the app layer code has already done so.
type NoResponse struct{}

// NewNoResponse constructs a no reponse value.
func NewNoResponse() NoResponse {
	return NoResponse{}
}

// Encode implements the Encoder interface.
func (NoResponse) Encode() ([]byte, string, error) {
	return nil, "", nil
}

// JSONResponse represents a JSON response with generic data type
type JSONResponse[T any] struct {
	Data   T
	Status int
}

func (j *JSONResponse[T]) Encode() ([]byte, string, error) {
	data, err := json.Marshal(j.Data)
	if err != nil {
		return nil, "", err
	}
	return data, "application/json; charset=utf-8", nil
}

func (j *JSONResponse[T]) HTTPStatus() int {
	if j.Status == 0 {
		return http.StatusOK
	}
	return j.Status
}

// Helper constructor functions
func NewJSONResponse[T any](data T) *JSONResponse[T] {
	return &JSONResponse[T]{Data: data}
}

func NewJSONResponseWithStatus[T any](data T, status int) *JSONResponse[T] {
	return &JSONResponse[T]{Data: data, Status: status}
}

// =============================================================================

type httpStatus interface {
	HTTPStatus() int
}

// Respond sends a response to the client.
func Respond(ctx context.Context, w http.ResponseWriter, resp Encoder) error {
	if _, ok := resp.(NoResponse); ok {
		return nil
	}

	// If the context has been canceled, it means the client is no longer
	// waiting for a response.
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return errors.New("client disconnected, do not send response")
		}
	}

	statusCode := http.StatusOK

	switch v := resp.(type) {
	case httpStatus:
		statusCode = v.HTTPStatus()

	case error:
		statusCode = http.StatusInternalServerError

	default:
		if resp == nil {
			statusCode = http.StatusNoContent
		}
	}

	// _, span := addSpan(ctx, "web.send.response", attribute.Int("status", statusCode))
	// defer span.End()

	if statusCode == http.StatusNoContent {
		w.WriteHeader(statusCode)
		return nil
	}

	data, contentType, err := resp.Encode()

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return fmt.Errorf("respond: encode: %w", err)
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(statusCode)

	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("respond: write: %w", err)
	}

	return nil
}
