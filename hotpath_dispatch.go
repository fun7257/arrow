package arrow

import (
	"net/http"
	"sync/atomic"
)

var (
	zeroMiddlewareRouterDispatches   atomic.Uint64
	zeroMiddlewarePipelineDispatches atomic.Uint64
)

func resetZeroMiddlewareDispatchCounters() {
	zeroMiddlewareRouterDispatches.Store(0)
	zeroMiddlewarePipelineDispatches.Store(0)
}

// serveZeroMiddlewareFromHTTP mirrors the router zero-middleware path for tests
// (newContext → recoverAndRelease → executeZeroMiddleware). Production router
// registration inlines the same steps; benchmarks without app.Use hit that path.
func serveZeroMiddlewareFromHTTP(w http.ResponseWriter, req *http.Request, handler HandlerFunc) {
	zeroMiddlewareRouterDispatches.Add(1)
	ctx := newContext(w, req)
	defer recoverAndRelease(ctx)
	executeZeroMiddleware(ctx, handler)
}