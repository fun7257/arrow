package arrow

// Group creates a child router with a path prefix and inherited middleware.
func (r *Router) Group(prefix string) *Router {
	return &Router{
		engine: r.engine,
		mux:    r.mux,
		prefix: joinPattern(r.prefix, prefix),
		pipe:   r.pipe.clone(),
	}
}