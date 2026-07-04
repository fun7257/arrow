package arrow

import (
	"net/http"
)

// Engine is the root HTTP application. It implements http.Handler.
type Engine struct {
	mux *http.ServeMux
}

// New creates a new Arrow application.
func New() *Router {
	mux := http.NewServeMux()
	return &Router{
		engine: &Engine{mux: mux},
		mux:    mux,
		pipe:   newPipeline(),
	}
}

// ServeHTTP dispatches the request to the underlying ServeMux.
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.mux.ServeHTTP(w, r)
}

// Handler returns the engine as an http.Handler.
func (r *Router) Handler() http.Handler {
	return r.engine
}

// ListenAndServe starts the HTTP server on addr.
func (r *Router) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, r.engine)
}

// Serve starts the server with the given http.Server.
func (r *Router) Serve(srv *http.Server) error {
	srv.Handler = r.engine
	return srv.ListenAndServe()
}