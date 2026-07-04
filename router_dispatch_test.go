package arrow_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fun7257/arrow"
)

// TestBenchHotPathUsesRouterZeroMiddlewareDispatch drives buildArrowApp (same as
// benchmarks) and asserts the request hits the router zero-mw path
// (executeZeroMiddleware + dispatch counter), not pipeline runNoMiddleware.
func TestBenchHotPathUsesRouterZeroMiddlewareDispatch(t *testing.T) {
	s := loadBenchScenario(t, "minimal.json")
	wantBody := s.Routes[0].Response
	req := benchRequest(probeRequest(s))

	arrow.ResetZeroMiddlewareDispatchCounters()
	h := buildArrowApp(s)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

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
		t.Fatalf("pipeline dispatches = %d, want 0 (bench path must not use runNoMiddleware)", got)
	}

	// Static corpus uses the same zero-mw registration path.
	arrow.ResetZeroMiddlewareDispatchCounters()
	sStatic := loadBenchScenario(t, "static.json")
	hStatic := buildArrowApp(sStatic)
	recStatic := httptest.NewRecorder()
	hStatic.ServeHTTP(recStatic, benchRequest(probeRequest(sStatic)))
	if arrow.ZeroMiddlewareRouterDispatches() != 1 {
		t.Fatalf("static router dispatches = %d, want 1", arrow.ZeroMiddlewareRouterDispatches())
	}
	if arrow.ZeroMiddlewarePipelineDispatches() != 0 {
		t.Fatalf("static pipeline dispatches = %d, want 0", arrow.ZeroMiddlewarePipelineDispatches())
	}

	var order []string
	app := arrow.New()
	app.GET(s.Routes[0].Pattern, func(c *arrow.Context) {
		c.After(func(c *arrow.Context) { order = append(order, "after") })
		order = append(order, "handler")
		c.Write([]byte(wantBody))
	})
	arrow.ResetZeroMiddlewareDispatchCounters()
	recAfter := httptest.NewRecorder()
	app.Handler().ServeHTTP(recAfter, req)
	if len(order) != 2 || order[0] != "handler" || order[1] != "after" {
		t.Fatalf("after order = %v, want [handler after]", order)
	}
	if arrow.ZeroMiddlewareRouterDispatches() != 1 {
		t.Fatalf("after router dispatches = %d, want 1", arrow.ZeroMiddlewareRouterDispatches())
	}

	// Middleware path must use pipeline.Run (runNoMiddleware when depth>0 still
	// goes through pipe.Run, but counters track runNoMiddleware only when
	// len(middlewares)==0 inside Run — with middleware, pipeline path differs).
	mwRan := false
	appMW := arrow.New()
	appMW.Use(func(c *arrow.Context) { mwRan = true })
	registerArrowRoutes(appMW, s.Routes)
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
	appAbort.GET(s.Routes[0].Pattern, func(c *arrow.Context) {
		c.Abort(http.StatusTeapot)
	})
	appAbort.Handler().ServeHTTP(recAbort, req)
	if recAbort.Code != http.StatusTeapot {
		t.Fatalf("abort status = %d, want %d", recAbort.Code, http.StatusTeapot)
	}

	recPanic := httptest.NewRecorder()
	appPanic := arrow.New()
	appPanic.GET(s.Routes[0].Pattern, func(c *arrow.Context) { panic("bench") })
	appPanic.Handler().ServeHTTP(recPanic, req)
	if recPanic.Code != http.StatusInternalServerError {
		t.Fatalf("panic status = %d, want %d", recPanic.Code, http.StatusInternalServerError)
	}
}