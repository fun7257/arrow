package target

import (
	"net/http"

	"github.com/fun7257/arrow"
)

// WriteEncoded writes a pre-built encoded target.
func WriteEncoded[T any](c *arrow.Context, e Encoded[T]) error {
	return Write(c, e)
}

// WriteJSON writes body as JSON with status.
func WriteJSON[T any](c *arrow.Context, status int, body T) error {
	return Write(c, JSON[T](status, body))
}

// WriteJSONIndent writes indented JSON.
func WriteJSONIndent[T any](c *arrow.Context, status int, body T, prefix, indent string) error {
	return Write(c, Encoded[T]{
		Status:  status,
		Encoder: IndentJSONEncoder[T]{Prefix: prefix, Indent: indent},
		Body:    body,
	})
}

// OK writes 200 JSON.
func OK[T any](c *arrow.Context, body T) error {
	return WriteJSON(c, http.StatusOK, body)
}

// Created writes 201 JSON.
func Created[T any](c *arrow.Context, body T) error {
	return WriteJSON(c, http.StatusCreated, body)
}

// Accepted writes 202 JSON.
func Accepted[T any](c *arrow.Context, body T) error {
	return WriteJSON(c, http.StatusAccepted, body)
}

// NoContent writes 204 with no body.
func NoContent(c *arrow.Context) error {
	return WriteStatus(c, http.StatusNoContent)
}

// AbortJSON aborts penetration after writing a JSON body.
func AbortJSON[T any](c *arrow.Context, status int, body T) error {
	return Abort(c, JSON[T](status, body))
}