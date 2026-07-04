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

	statusW *statusWriter
	aborted bool
	code    int
	written bool
	afters  []HandlerFunc
	keys    map[string]any
}

func newContext(w http.ResponseWriter, r *http.Request) *Context {
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	return &Context{
		Writer:  wrapResponseWriter(sw),
		Request: r,
		statusW: sw,
	}
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

// multiWriter embeds optional interfaces only when the underlying writer
// provides them. Nil embedded interfaces are omitted from the method set,
// matching net/http type-assertion behavior.
type multiWriter struct {
	*statusWriter
	http.Flusher
	http.Hijacker
	http.Pusher
	readerFrom io.ReaderFrom
}

func (m *multiWriter) Unwrap() http.ResponseWriter {
	return m.statusWriter
}

func (m *multiWriter) ReadFrom(r io.Reader) (int64, error) {
	if m.readerFrom != nil {
		return m.readerFrom.ReadFrom(r)
	}
	return io.Copy(m.statusWriter, r)
}

func wrapResponseWriter(sw *statusWriter) http.ResponseWriter {
	w := sw.ResponseWriter
	mw := &multiWriter{statusWriter: sw}
	if f, ok := w.(http.Flusher); ok {
		mw.Flusher = f
	}
	if h, ok := w.(http.Hijacker); ok {
		mw.Hijacker = h
	}
	if p, ok := w.(http.Pusher); ok {
		mw.Pusher = p
	}
	if rf, ok := w.(io.ReaderFrom); ok {
		mw.readerFrom = &readerFromDelegator{inner: sw, delegate: rf}
	}
	return mw
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
	if c.statusW != nil {
		return c.statusW.status
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