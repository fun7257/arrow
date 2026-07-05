package arrow

import "net/http"

// groupRoutes is a child router scope returned by Group.
// Use has a pointer receiver; assign the Group result before calling Use
// (e.g. api := app.Group(prefix); api.Use(mw)).
type groupRoutes struct {
	r *Router
}

// Group creates a child router with a path prefix and inherited middleware.
func (r *Router) Group(prefix string) groupRoutes {
	return groupRoutes{r: &Router{
		engine: r.engine,
		mux:    r.mux,
		prefix: joinPattern(r.prefix, prefix),
		pipe:   r.pipe.clone(),
	}}
}

// Use registers one middleware on this group scope.
func (g *groupRoutes) Use(mw HandlerFunc) {
	g.r.Use(mw)
}

func (g groupRoutes) Group(prefix string) groupRoutes {
	return g.r.Group(prefix)
}

func (g groupRoutes) GET(pattern string, handler HandlerFunc) {
	g.r.GET(pattern, handler)
}

func (g groupRoutes) POST(pattern string, handler HandlerFunc) {
	g.r.POST(pattern, handler)
}

func (g groupRoutes) PUT(pattern string, handler HandlerFunc) {
	g.r.PUT(pattern, handler)
}

func (g groupRoutes) DELETE(pattern string, handler HandlerFunc) {
	g.r.DELETE(pattern, handler)
}

func (g groupRoutes) PATCH(pattern string, handler HandlerFunc) {
	g.r.PATCH(pattern, handler)
}

func (g groupRoutes) HEAD(pattern string, handler HandlerFunc) {
	g.r.HEAD(pattern, handler)
}

func (g groupRoutes) OPTIONS(pattern string, handler HandlerFunc) {
	g.r.OPTIONS(pattern, handler)
}

func (g groupRoutes) Any(pattern string, handler HandlerFunc) {
	g.r.Any(pattern, handler)
}

func (g groupRoutes) Handle(method, pattern string, handler HandlerFunc) {
	g.r.Handle(method, pattern, handler)
}

func (g groupRoutes) HandleHTTP(pattern string, h http.Handler) {
	g.r.HandleHTTP(pattern, h)
}

func (g groupRoutes) HandleHTTPMethod(method, pattern string, h http.Handler) {
	g.r.HandleHTTPMethod(method, pattern, h)
}

func (g groupRoutes) Mux() *http.ServeMux {
	return g.r.Mux()
}
