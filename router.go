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

func (r *Router) register(method, pattern string, handler HandlerFunc) {
	fullPattern := joinPattern(r.prefix, pattern)
	muxPattern := method + " " + fullPattern

	pipe := r.pipe
	wrapped := func(w http.ResponseWriter, req *http.Request) {
		ctx := newContext(w, req)
		pipe.Run(ctx, handler)
	}
	r.mux.HandleFunc(muxPattern, wrapped)
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

// Handle registers a route with an arbitrary HTTP method.
func (r *Router) Handle(method, pattern string, handler HandlerFunc) {
	r.register(method, pattern, handler)
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