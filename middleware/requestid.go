package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/fun7257/arrow"
)

const (
	// RequestIDKey is the Context key for the request ID.
	RequestIDKey = "arrow.request_id"

	headerRequestID = "X-Request-ID"
)

// RequestID assigns a unique ID to each request.
func RequestID() arrow.HandlerFunc {
	return func(c *arrow.Context) {
		id := c.Request.Header.Get(headerRequestID)
		if id == "" {
			id = newRequestID()
		}
		c.Set(RequestIDKey, id)
		c.Writer.Header().Set(headerRequestID, id)
	}
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b[:])
}