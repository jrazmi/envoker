package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Encoder defines behavior that can encode a data model and provide the content type
type Encoder interface {
	Encode() (data []byte, contentType string, err error)
}

// httpStatus allows responses to specify HTTP status codes
type httpStatus interface {
	HTTPStatus() int
}

// Respond sends a response to the client with smart encoding
func Respond(ctx context.Context, w http.ResponseWriter, resp Encoder) error {
	if _, ok := resp.(NoResponse); ok {
		return nil
	}

	// Check for canceled context
	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			return errors.New("client disconnected, do not send response")
		}
	}

	// Handle redirects specially
	if redirect, ok := resp.(Redirect); ok {
		w.Header().Set("Location", redirect.URL)
		w.WriteHeader(redirect.HTTPStatus())
		return nil
	}

	// Determine status code
	statusCode := http.StatusOK
	if statusResp, ok := resp.(httpStatus); ok {
		statusCode = statusResp.HTTPStatus()
	}

	// TODO - different content type error handlers.
	// Handle errors specially
	if err, ok := resp.(error); ok {
		statusCode = http.StatusInternalServerError
		// Convert error to JSON response
		errorData := map[string]string{"error": err.Error()}
		jsonData, _ := json.Marshal(errorData)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(statusCode)
		w.Write(jsonData)
		return nil
	}

	// Handle nil response
	if resp == nil {
		w.WriteHeader(http.StatusNoContent)
		return nil
	}

	// Standard encoding path
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

// JSON automatically encodes any data as JSON
type JSON struct {
	Data   interface{}
	Status int
}

func NewJSON(data interface{}) JSON {
	return JSON{Data: data, Status: http.StatusOK}
}

func NewJSONWithStatus(data interface{}, status int) JSON {
	return JSON{Data: data, Status: status}
}

func (j JSON) Encode() ([]byte, string, error) {
	data, err := json.Marshal(j.Data)
	if err != nil {
		return nil, "", err
	}
	return data, "application/json; charset=utf-8", nil
}

func (j JSON) HTTPStatus() int {
	return j.Status
}

// Text for plain text responses
type Text struct {
	Data   string
	Status int
}

func NewText(data string) Text {
	return Text{Data: data, Status: http.StatusOK}
}

func NewTextWithStatus(data string, status int) Text {
	return Text{Data: data, Status: status}
}

func (t Text) Encode() ([]byte, string, error) {
	return []byte(t.Data), "text/plain; charset=utf-8", nil
}

func (t Text) HTTPStatus() int {
	return t.Status
}

// HTML for HTML responses
type HTML struct {
	Data   string
	Status int
}

func NewHTML(data string) HTML {
	return HTML{Data: data, Status: http.StatusOK}
}

func (h HTML) Encode() ([]byte, string, error) {
	return []byte(h.Data), "text/html; charset=utf-8", nil
}

func (h HTML) HTTPStatus() int {
	return h.Status
}

// Raw for custom responses (you control everything)
type Raw struct {
	Data        []byte
	ContentType string
	Status      int
}

func NewRaw(data []byte, contentType string) Raw {
	return Raw{Data: data, ContentType: contentType, Status: http.StatusOK}
}

func (r Raw) Encode() ([]byte, string, error) {
	return r.Data, r.ContentType, nil
}

func (r Raw) HTTPStatus() int {
	return r.Status
}

// NoResponse tells Respond to not send anything
type NoResponse struct{}

func NewNoResponse() NoResponse {
	return NoResponse{}
}

func (NoResponse) Encode() ([]byte, string, error) {
	return nil, "", nil
}

// Redirect for HTTP redirects
type Redirect struct {
	URL    string
	Status int
}

func NewRedirect(url string) Redirect {
	return Redirect{URL: url, Status: http.StatusFound}
}

func NewRedirectWithStatus(url string, status int) Redirect {
	return Redirect{URL: url, Status: status}
}

func (r Redirect) Encode() ([]byte, string, error) {
	// Special case - redirect doesn't return data, just sets headers
	return []byte{}, "", nil
}

func (r Redirect) HTTPStatus() int {
	return r.Status
}
