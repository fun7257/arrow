package target

import (
	"fmt"
	"io"
	"net/http"

	"github.com/fun7257/arrow"
)

// WriteStream writes a streaming body with status and contentType.
// Headers and status are committed before fn runs; callback errors cannot change the status code.
func WriteStream(c *arrow.Context, status int, contentType string, fn func(w io.Writer) error) error {
	return Write(c, streamTarget{status: status, contentType: contentType, fn: fn})
}

// WriteStreamReader copies r to the response with status and contentType.
func WriteStreamReader(c *arrow.Context, status int, contentType string, r io.Reader) error {
	return WriteStream(c, status, contentType, func(w io.Writer) error {
		_, err := io.Copy(w, r)
		return err
	})
}

type streamTarget struct {
	status      int
	contentType string
	fn          func(w io.Writer) error
}

func (s streamTarget) StatusCode() int {
	if s.status == 0 {
		return http.StatusOK
	}
	return s.status
}

func (s streamTarget) Respond(c *arrow.Context) error {
	if c.Written() {
		return nil
	}
	if s.contentType != "" {
		c.Writer.Header().Set("Content-Type", s.contentType)
	}
	c.WriteHeader(s.StatusCode())
	return s.fn(c.Writer)
}

// EventWriter writes Server-Sent Events.
type EventWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// WriteSSE starts an SSE response and runs fn with an EventWriter.
// SSE must be the sole response for the request.
// Headers and status are committed before fn runs; callback errors cannot change the status code.
func WriteSSE(c *arrow.Context, fn func(w *EventWriter) error) error {
	return Write(c, sseTarget{fn: fn})
}

type sseTarget struct {
	fn func(w *EventWriter) error
}

func (s sseTarget) StatusCode() int {
	return http.StatusOK
}

func (s sseTarget) Respond(c *arrow.Context) error {
	if c.Written() {
		return nil
	}
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		return fmt.Errorf("target: streaming unsupported")
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.WriteHeader(http.StatusOK)
	return s.fn(&EventWriter{w: c.Writer, flusher: flusher})
}

// Event writes a named SSE event with data.
func (e *EventWriter) Event(name, data string) error {
	if name != "" {
		if _, err := fmt.Fprintf(e.w, "event: %s\n", name); err != nil {
			return err
		}
	}
	for _, line := range splitLines(data) {
		if _, err := fmt.Fprintf(e.w, "data: %s\n", line); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(e.w, "\n"); err != nil {
		return err
	}
	e.flusher.Flush()
	return nil
}

// Data writes an SSE data-only event.
func (e *EventWriter) Data(data string) error {
	return e.Event("", data)
}

func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}