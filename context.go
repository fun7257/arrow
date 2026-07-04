package arrow

import "net/http"

// HandlerFunc is a request handler in the Arrow penetration model.
type HandlerFunc func(*Context)

// Context carries a request through the linear middleware pipeline.
type Context struct {
	Writer  http.ResponseWriter
	Request *http.Request

	aborted bool
	code    int
	written bool
	afters  []HandlerFunc
	keys    map[string]any
}

func newContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		Writer:  &statusWriter{ResponseWriter: w, status: http.StatusOK},
		Request: r,
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
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
	if sw, ok := c.Writer.(*statusWriter); ok {
		return sw.status
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