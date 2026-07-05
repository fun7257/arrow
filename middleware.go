package arrow

// Use registers one middleware on this router. Middleware applies to all routes
// registered on this router and its child groups created after Use is called.
func (r *Router) Use(mw HandlerFunc) {
	r.pipe.Use(mw)
}
