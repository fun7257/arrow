package target

import (
	"net/http"

	"github.com/fun7257/arrow"
)

// Envelope is a generic API response wrapper.
type Envelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    T      `json:"data"`
}

// OKEnvelope writes a successful envelope response.
func OKEnvelope[T any](c *arrow.Context, data T) error {
	return WriteJSON(c, http.StatusOK, Envelope[T]{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

// ErrorEnvelope writes an error envelope response.
func ErrorEnvelope(c *arrow.Context, status int, code int, message string) error {
	return WriteJSON(c, status, Envelope[any]{
		Code:    code,
		Message: message,
	})
}