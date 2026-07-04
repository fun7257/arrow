package arrow

import (
	"io"
	"net/http"
)

// HandlerFunc is a request handler in the Arrow penetration model.
type HandlerFunc func(*Context)

// Context carries a request through the linear middleware pipeline.
type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request

	sw statusWriter

	aborted bool
	code    int
	written bool
	afters  []HandlerFunc
	keys    map[string]any

	wrapMask int8
	wrapPtr  any
}

func newContext(w http.ResponseWriter, r *http.Request) *Context {
	c := acquireContext()
	c.Request = r
	c.sw = statusWriter{ResponseWriter: w, status: http.StatusOK}
	c.Writer = wrapResponseWriter(&c.sw, c)
	return c
}

type statusWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (w *statusWriter) WriteHeader(code int) {
	if w.written {
		return
	}
	w.status = code
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap returns the underlying ResponseWriter for http.ResponseController.
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// readerFromDelegator ensures WriteHeader runs before ReadFrom delegation.
type readerFromDelegator struct {
	inner    *statusWriter
	delegate io.ReaderFrom
}

func (d *readerFromDelegator) ReadFrom(r io.Reader) (int64, error) {
	if !d.inner.written {
		d.inner.WriteHeader(http.StatusOK)
	}
	return d.delegate.ReadFrom(r)
}

// After registers post-handler logic. Callbacks run in registration order
// after the route handler completes.
func (c *Context) After(fn HandlerFunc) {
	c.afters = append(c.afters, fn)
}

// Abort stops penetration: remaining Pre middleware and the route handler
// are skipped. Already registered After callbacks still run.
func (c *Context) Abort(code int) {
	if c.aborted {
		return
	}
	c.aborted = true
	c.code = code
	if !c.written {
		c.Writer.WriteHeader(code)
		c.written = true
	}
}

// Penetrate is an explicit no-op marking penetration through the current layer.
// Returning from middleware is equivalent.
func (c *Context) Penetrate() {}

// IsAborted reports whether the request was aborted.
func (c *Context) IsAborted() bool {
	return c.aborted
}

// Status returns the response status code.
func (c *Context) Status() int {
	if c.aborted {
		return c.code
	}
	if c.sw.ResponseWriter != nil {
		return c.sw.status
	}
	return 0
}

// Set stores a value scoped to this request.
func (c *Context) Set(key string, value any) {
	if c.keys == nil {
		c.keys = make(map[string]any)
	}
	c.keys[key] = value
}

// Get retrieves a value stored on this request.
func (c *Context) Get(key string) (any, bool) {
	if c.keys == nil {
		return nil, false
	}
	v, ok := c.keys[key]
	return v, ok
}

// WriteHeader records the response status.
func (c *Context) WriteHeader(code int) {
	c.Writer.WriteHeader(code)
	c.written = true
}

// Write writes response data.
func (c *Context) Write(b []byte) (int, error) {
	return c.Writer.Write(b)
}