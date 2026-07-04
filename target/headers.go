package target

import (
	"net/http"

	"github.com/fun7257/arrow"
)

// SetHeader sets a single response header.
func SetHeader(c *arrow.Context, key, value string) {
	c.Writer.Header().Set(key, value)
}

// SetHeaders sets multiple response headers.
func SetHeaders(c *arrow.Context, headers map[string]string) {
	for k, v := range headers {
		c.Writer.Header().Set(k, v)
	}
}

// SetCookie sets a response cookie.
func SetCookie(c *arrow.Context, cookie *http.Cookie) {
	http.SetCookie(c.Writer, cookie)
}

// WriteWithHeaders writes t after applying headers.
func WriteWithHeaders(c *arrow.Context, t Target, headers map[string]string) error {
	for k, v := range headers {
		c.Writer.Header().Set(k, v)
	}
	return Write(c, t)
}

// WriteStatus writes only an HTTP status code.
func WriteStatus(c *arrow.Context, status int) error {
	return Write(c, statusOnly{status: status})
}

type statusOnly struct {
	status int
}

func (s statusOnly) StatusCode() int {
	return s.status
}

func (s statusOnly) Respond(c *arrow.Context) error {
	if c.Written() {
		return nil
	}
	c.WriteHeader(s.status)
	return nil
}