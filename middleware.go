package arrow

// Use registers middleware on this router. Middleware applies to all routes
// registered on this router and its child groups created after Use is called.
func (r *Router) Use(middleware ...HandlerFunc) *Router {
	for _, mw := range middleware {
		r.pipe.Use(mw)
	}
	return r
}