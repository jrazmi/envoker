package web

import "encoding/json"

// Error response type that implements Encoder
type ErrorResponse struct {
	Error string `json:"error"`
}

func NewError(msg string) ErrorResponse {
	return ErrorResponse{Error: msg}
}

func (e ErrorResponse) Encode() ([]byte, string, error) {
	data, err := json.Marshal(e)
	return data, "application/json", err
}

func (e ErrorResponse) HTTPStatus() int {
	return 500
}
