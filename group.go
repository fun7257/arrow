package arrow

// groupRoutes is a child router scope. Use has a pointer receiver; assign the
// Group result before calling Use (e.g. api := app.Group(prefix); api.Use(mw)).
type groupRoutes struct {
	*Router
}

// Group creates a child router with a path prefix and inherited middleware.
func (r *Router) Group(prefix string) groupRoutes {
	return groupRoutes{&Router{
		engine: r.engine,
		mux:    r.mux,
		prefix: joinPattern(r.prefix, prefix),
		pipe:   r.pipe.clone(),
	}}
}

// Use registers one middleware on this group scope.
func (g *groupRoutes) Use(mw HandlerFunc) {
	g.Router.Use(mw)
}