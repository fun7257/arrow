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

// serveZeroMiddlewareFromHTTP is the unified zero-middleware entry for router
// registration. Benchmarks without app.Use hit this path, not runNoMiddleware.
func serveZeroMiddlewareFromHTTP(w http.ResponseWriter, req *http.Request, handler HandlerFunc) {
	zeroMiddlewareRouterDispatches.Add(1)
	ctx := newContext(w, req)
	defer recoverAndRelease(ctx)
	executeZeroMiddleware(ctx, handler)
}