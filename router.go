package arrow

import (
	"net/http"
	"strings"
)

// Router registers routes and middleware. The root router created by New
// also serves as the application entry point.
type Router struct {
	engine *Engine
	mux    *http.ServeMux
	prefix string
	pipe   *pipeline
}

func (r *Router) muxPattern(method, pattern string) string {
	fullPattern := joinPattern(r.prefix, pattern)
	if method == "" {
		return fullPattern
	}
	return method + " " + fullPattern
}

func (r *Router) register(method, pattern string, handler HandlerFunc) {
	muxPattern := r.muxPattern(method, pattern)
	if len(r.pipe.middlewares) == 0 {
		// Bench scenarios (minimal/static/…) register without global middleware;
		// the hot path is this inline closure, not pipeline.Run.
		wrapped := func(w http.ResponseWriter, req *http.Request) {
			ctx := newContext(w, req)
			defer recoverAndRelease(ctx)
			handler(ctx)
			for _, after := range ctx.afters {
				after(ctx)
			}
		}
		r.mux.HandleFunc(muxPattern, wrapped)
		return
	}
	pipe := r.pipe
	wrapped := func(w http.ResponseWriter, req *http.Request) {
		ctx := newContext(w, req)
		pipe.Run(ctx, handler)
	}
	r.mux.HandleFunc(muxPattern, wrapped)
}

func (r *Router) registerHTTP(method, pattern string, h http.Handler) {
	fullPattern := joinPattern(r.prefix, pattern)
	handler := h
	muxPattern := r.muxPattern(method, pattern)

	// Go 1.22+ ServeMux requires wildcard patterns for subtree handlers
	// such as http.FileServer. StripPrefix is applied automatically.
	if strings.HasSuffix(fullPattern, "/") && !strings.Contains(fullPattern, "{") {
		dir := strings.TrimSuffix(fullPattern, "/")
		handler = http.StripPrefix(fullPattern, h)
		if method == "" {
			muxPattern = dir + "/{path...}"
		} else {
			muxPattern = method + " " + dir + "/{path...}"
		}
	}

	if len(r.pipe.middlewares) == 0 {
		wrapped := func(w http.ResponseWriter, req *http.Request) {
			ctx := newContext(w, req)
			defer recoverAndRelease(ctx)
			handler.ServeHTTP(ctx.Writer, ctx.Request)
			for _, after := range ctx.afters {
				after(ctx)
			}
		}
		r.mux.Handle(muxPattern, http.HandlerFunc(wrapped))
		return
	}
	pipe := r.pipe
	wrapped := func(w http.ResponseWriter, req *http.Request) {
		ctx := newContext(w, req)
		pipe.Run(ctx, func(c *Context) {
			handler.ServeHTTP(c.Writer, c.Request)
		})
	}
	r.mux.Handle(muxPattern, http.HandlerFunc(wrapped))
}

// GET registers a GET route.
func (r *Router) GET(pattern string, handler HandlerFunc) {
	r.register(http.MethodGet, pattern, handler)
}

// POST registers a POST route.
func (r *Router) POST(pattern string, handler HandlerFunc) {
	r.register(http.MethodPost, pattern, handler)
}

// PUT registers a PUT route.
func (r *Router) PUT(pattern string, handler HandlerFunc) {
	r.register(http.MethodPut, pattern, handler)
}

// DELETE registers a DELETE route.
func (r *Router) DELETE(pattern string, handler HandlerFunc) {
	r.register(http.MethodDelete, pattern, handler)
}

// PATCH registers a PATCH route.
func (r *Router) PATCH(pattern string, handler HandlerFunc) {
	r.register(http.MethodPatch, pattern, handler)
}

// HEAD registers a HEAD route.
func (r *Router) HEAD(pattern string, handler HandlerFunc) {
	r.register(http.MethodHead, pattern, handler)
}

// OPTIONS registers an OPTIONS route.
func (r *Router) OPTIONS(pattern string, handler HandlerFunc) {
	r.register(http.MethodOptions, pattern, handler)
}

// Any registers a route that matches all HTTP methods.
func (r *Router) Any(pattern string, handler HandlerFunc) {
	r.register("", pattern, handler)
}

// Handle registers a route with an arbitrary HTTP method.
// Pass an empty method to match all methods.
func (r *Router) Handle(method, pattern string, handler HandlerFunc) {
	r.register(method, pattern, handler)
}

// HandleHTTP registers a standard library http.Handler for all methods.
func (r *Router) HandleHTTP(pattern string, h http.Handler) {
	r.registerHTTP("", pattern, h)
}

// HandleHTTPMethod registers a standard library http.Handler for a specific method.
func (r *Router) HandleHTTPMethod(method, pattern string, h http.Handler) {
	r.registerHTTP(method, pattern, h)
}

// Mux returns the underlying http.ServeMux for advanced use.
func (r *Router) Mux() *http.ServeMux {
	return r.mux
}

func joinPattern(prefix, pattern string) string {
	if prefix == "" {
		return pattern
	}
	if pattern == "" {
		return prefix
	}
	return strings.TrimSuffix(prefix, "/") + "/" + strings.TrimPrefix(pattern, "/")
}