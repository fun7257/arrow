package arrow_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fun7257/arrow"
)

// TestBenchHotPathUsesRouterZeroMiddlewareDispatch asserts zero-middleware routes hit
// the router inline path (executeZeroMiddleware + dispatch counter), not pipeline
// runNoMiddleware.
func TestBenchHotPathUsesRouterZeroMiddlewareDispatch(t *testing.T) {
	const wantBody = "pong"
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	app := arrow.New()
	app.GET("/ping", func(c *arrow.Context) {
		c.Write([]byte(wantBody))
	})

	arrow.ResetZeroMiddlewareDispatchCounters()
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != wantBody {
		t.Fatalf("body = %q, want %q", got, wantBody)
	}
	if got := arrow.ZeroMiddlewareRouterDispatches(); got != 1 {
		t.Fatalf("router dispatches = %d, want 1 (router zero-mw path)", got)
	}
	if got := arrow.ZeroMiddlewarePipelineDispatches(); got != 0 {
		t.Fatalf("pipeline dispatches = %d, want 0 (zero-mw path must not use runNoMiddleware)", got)
	}

	// Multi-route registration uses the same zero-mw path.
	staticRoutes := []struct {
		pattern  string
		response string
	}{
		{"/api/v1/users", "users"},
		{"/api/v1/orgs", "orgs"},
		{"/api/v1/repos", "repos"},
		{"/api/v1/gists", "gists"},
		{"/api/v1/events", "events"},
		{"/api/v1/feeds", "feeds"},
		{"/api/v1/notifications", "notifications"},
		{"/api/v1/marketplace", "marketplace"},
		{"/api/v1/meta", "meta"},
		{"/api/v1/rate_limit", "rate_limit"},
		{"/api/v1/emojis", "emojis"},
		{"/api/v1/search", "search"},
	}
	staticApp := arrow.New()
	for _, rt := range staticRoutes {
		body := []byte(rt.response)
		pattern := rt.pattern
		staticApp.GET(pattern, func(c *arrow.Context) {
			c.Write(body)
		})
	}
	staticReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)

	arrow.ResetZeroMiddlewareDispatchCounters()
	recStatic := httptest.NewRecorder()
	staticApp.Handler().ServeHTTP(recStatic, staticReq)
	if arrow.ZeroMiddlewareRouterDispatches() != 1 {
		t.Fatalf("static router dispatches = %d, want 1", arrow.ZeroMiddlewareRouterDispatches())
	}
	if arrow.ZeroMiddlewarePipelineDispatches() != 0 {
		t.Fatalf("static pipeline dispatches = %d, want 0", arrow.ZeroMiddlewarePipelineDispatches())
	}

	var order []string
	appAfter := arrow.New()
	appAfter.GET("/ping", func(c *arrow.Context) {
		c.After(func(c *arrow.Context) { order = append(order, "after") })
		order = append(order, "handler")
		c.Write([]byte(wantBody))
	})
	arrow.ResetZeroMiddlewareDispatchCounters()
	recAfter := httptest.NewRecorder()
	appAfter.Handler().ServeHTTP(recAfter, req)
	if len(order) != 2 || order[0] != "handler" || order[1] != "after" {
		t.Fatalf("after order = %v, want [handler after]", order)
	}
	if arrow.ZeroMiddlewareRouterDispatches() != 1 {
		t.Fatalf("after router dispatches = %d, want 1", arrow.ZeroMiddlewareRouterDispatches())
	}

	mwRan := false
	appMW := arrow.New()
	appMW.Use(func(c *arrow.Context) { mwRan = true })
	appMW.GET("/ping", func(c *arrow.Context) {
		c.Write([]byte(wantBody))
	})
	arrow.ResetZeroMiddlewareDispatchCounters()
	recMW := httptest.NewRecorder()
	appMW.Handler().ServeHTTP(recMW, req)
	if !mwRan {
		t.Fatal("sanity: middleware must run when app.Use is called")
	}
	if arrow.ZeroMiddlewareRouterDispatches() != 0 {
		t.Fatalf("middleware router dispatches = %d, want 0", arrow.ZeroMiddlewareRouterDispatches())
	}

	recAbort := httptest.NewRecorder()
	appAbort := arrow.New()
	appAbort.GET("/ping", func(c *arrow.Context) {
		c.Abort(http.StatusTeapot)
	})
	appAbort.Handler().ServeHTTP(recAbort, req)
	if recAbort.Code != http.StatusTeapot {
		t.Fatalf("abort status = %d, want %d", recAbort.Code, http.StatusTeapot)
	}

	recPanic := httptest.NewRecorder()
	appPanic := arrow.New()
	appPanic.GET("/ping", func(c *arrow.Context) { panic("dispatch") })
	appPanic.Handler().ServeHTTP(recPanic, req)
	if recPanic.Code != http.StatusInternalServerError {
		t.Fatalf("panic status = %d, want %d", recPanic.Code, http.StatusInternalServerError)
	}
}