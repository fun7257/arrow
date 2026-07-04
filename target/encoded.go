package target

import (
	"bytes"
	"net/http"

	"github.com/fun7257/arrow"
)

// Encoded is a typed encoded response target.
type Encoded[T any] struct {
	Status  int
	Encoder Encoder[T]
	Body    T
	Headers map[string]string
	Cookies []*http.Cookie
}

func (e Encoded[T]) StatusCode() int {
	return e.Status
}

// Respond writes headers, cookies, status, and encoded body.
// Encoding is buffered so headers are not committed until encode succeeds.
// The full encoded body is held in memory before write; use WriteStream for large payloads.
func (e Encoded[T]) Respond(c *arrow.Context) error {
	if c.Written() {
		return nil
	}
	if e.Status == http.StatusNoContent {
		for k, v := range e.Headers {
			c.Writer.Header().Set(k, v)
		}
		for _, cookie := range e.Cookies {
			http.SetCookie(c.Writer, cookie)
		}
		c.WriteHeader(e.Status)
		return nil
	}
	var buf bytes.Buffer
	if err := e.Encoder.Encode(&buf, e.Body); err != nil {
		if Default.OnEncodeError != nil {
			Default.OnEncodeError(c, err)
		}
		return err
	}
	for k, v := range e.Headers {
		c.Writer.Header().Set(k, v)
	}
	for _, cookie := range e.Cookies {
		http.SetCookie(c.Writer, cookie)
	}
	if ct := e.Encoder.ContentType(); ct != "" {
		c.Writer.Header().Set("Content-Type", ct)
	}
	c.WriteHeader(e.Status)
	_, err := c.Write(buf.Bytes())
	return err
}

// JSON returns a JSON-encoded target.
func JSON[T any](status int, body T) Target {
	return Encoded[T]{
		Status:  status,
		Encoder: JSONEncoder[T]{},
		Body:    body,
	}
}

// XML returns an XML-encoded target.
func XML[T any](status int, body T) Target {
	return Encoded[T]{
		Status:  status,
		Encoder: XMLEncoder[T]{},
		Body:    body,
	}
}