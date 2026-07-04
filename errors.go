package arrow

import "errors"

var (
	// ErrServerClosed is returned when the server has been shut down.
	ErrServerClosed = errors.New("arrow: server closed")
)