package middleware

import (
	"log"
	"time"

	"github.com/fun7257/arrow"
)

// Logger logs each request with method, path, status and duration.
func Logger() arrow.HandlerFunc {
	return func(c *arrow.Context) {
		start := time.Now()

		c.After(func(c *arrow.Context) {
			status := c.Status()
			if status == 0 {
				status = 200
			}
			id, _ := c.Get(RequestIDKey)
			log.Printf("[%v] %s %s %d %v",
				id, c.Request.Method, c.Request.URL.Path, status, time.Since(start))
		})
	}
}