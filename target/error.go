package target

import (
	"net/http"

	"github.com/fun7257/arrow"
)

// Error is a simple JSON error payload.
// Message is serialized as the JSON key "error".
type Error struct {
	Message string `json:"error"`
}

// WriteError writes a JSON error response.
func WriteError(c *arrow.Context, status int, message string) error {
	return WriteJSON(c, status, Error{Message: message})
}

// BadRequest writes 400 JSON error.
func BadRequest(c *arrow.Context, message string) error {
	return WriteError(c, http.StatusBadRequest, message)
}

// Unauthorized writes 401 JSON error.
func Unauthorized(c *arrow.Context, message string) error {
	return WriteError(c, http.StatusUnauthorized, message)
}

// Forbidden writes 403 JSON error.
func Forbidden(c *arrow.Context, message string) error {
	return WriteError(c, http.StatusForbidden, message)
}

// NotFound writes 404 JSON error.
func NotFound(c *arrow.Context, message string) error {
	return WriteError(c, http.StatusNotFound, message)
}

// MethodNotAllowed writes 405 JSON error.
func MethodNotAllowed(c *arrow.Context, message string) error {
	return WriteError(c, http.StatusMethodNotAllowed, message)
}

// Conflict writes 409 JSON error.
func Conflict(c *arrow.Context, message string) error {
	return WriteError(c, http.StatusConflict, message)
}

// UnprocessableEntity writes 422 JSON error.
func UnprocessableEntity(c *arrow.Context, message string) error {
	return WriteError(c, http.StatusUnprocessableEntity, message)
}

// TooManyRequests writes 429 JSON error.
func TooManyRequests(c *arrow.Context, message string) error {
	return WriteError(c, http.StatusTooManyRequests, message)
}

// InternalError writes 500 JSON error.
func InternalError(c *arrow.Context, message string) error {
	return WriteError(c, http.StatusInternalServerError, message)
}

// NotImplemented writes 501 JSON error.
func NotImplemented(c *arrow.Context, message string) error {
	return WriteError(c, http.StatusNotImplemented, message)
}

// ServiceUnavailable writes 503 JSON error.
func ServiceUnavailable(c *arrow.Context, message string) error {
	return WriteError(c, http.StatusServiceUnavailable, message)
}

// AbortWith aborts penetration after writing t.
func AbortWith(c *arrow.Context, t Target) error {
	return Abort(c, t)
}

// AbortJSON aborts with a JSON body.
func AbortJSON[T any](c *arrow.Context, status int, body T) error {
	return Abort(c, JSON[T](status, body))
}

// AbortError aborts with a JSON error payload.
func AbortError(c *arrow.Context, status int, message string) error {
	return Abort(c, JSON(status, Error{Message: message}))
}

// AbortUnauthorized aborts with 401 JSON error.
func AbortUnauthorized(c *arrow.Context, message string) error {
	return AbortError(c, http.StatusUnauthorized, message)
}

// AbortForbidden aborts with 403 JSON error.
func AbortForbidden(c *arrow.Context, message string) error {
	return AbortError(c, http.StatusForbidden, message)
}

// AbortNotFound aborts with 404 JSON error.
func AbortNotFound(c *arrow.Context, message string) error {
	return AbortError(c, http.StatusNotFound, message)
}

// AbortBadRequest aborts with 400 JSON error.
func AbortBadRequest(c *arrow.Context, message string) error {
	return AbortError(c, http.StatusBadRequest, message)
}

// AbortMethodNotAllowed aborts with 405 JSON error.
func AbortMethodNotAllowed(c *arrow.Context, message string) error {
	return AbortError(c, http.StatusMethodNotAllowed, message)
}

// AbortConflict aborts with 409 JSON error.
func AbortConflict(c *arrow.Context, message string) error {
	return AbortError(c, http.StatusConflict, message)
}

// AbortUnprocessableEntity aborts with 422 JSON error.
func AbortUnprocessableEntity(c *arrow.Context, message string) error {
	return AbortError(c, http.StatusUnprocessableEntity, message)
}

// AbortTooManyRequests aborts with 429 JSON error.
func AbortTooManyRequests(c *arrow.Context, message string) error {
	return AbortError(c, http.StatusTooManyRequests, message)
}

// AbortInternalError aborts with 500 JSON error.
func AbortInternalError(c *arrow.Context, message string) error {
	return AbortError(c, http.StatusInternalServerError, message)
}

// AbortNotImplemented aborts with 501 JSON error.
func AbortNotImplemented(c *arrow.Context, message string) error {
	return AbortError(c, http.StatusNotImplemented, message)
}

// AbortServiceUnavailable aborts with 503 JSON error.
func AbortServiceUnavailable(c *arrow.Context, message string) error {
	return AbortError(c, http.StatusServiceUnavailable, message)
}