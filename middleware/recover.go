package middleware

import "github.com/fun7257/arrow"

// Recover marks the pipeline for panic recovery. The Arrow pipeline
// automatically recovers panics and returns 500 when this middleware
// is registered.
func Recover() arrow.HandlerFunc {
	return func(c *arrow.Context) {}
}