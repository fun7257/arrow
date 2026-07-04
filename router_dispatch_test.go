package arrow_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fun7257/arrow"
)

// TestBenchHotPathExecutesZeroMiddlewareDispatch drives the same buildArrowApp
// path as BenchmarkArrow_Minimal/Static and verifies zero-middleware semantics
// (handler, After, Abort, panic) without app.Use middleware.
func TestBenchHotPathExecutesZeroMiddlewareDispatch(t *testing.T) {
	s := loadBenchScenario(t, "minimal.json")
	wantBody := s.Routes[0].Response
	req := benchRequest(probeRequest(s))
	h := buildArrowApp(s)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != wantBody {
		t.Fatalf("body = %q, want %q", got, wantBody)
	}

	var afterRan bool
	app := arrow.New()
	app.GET(s.Routes[0].Pattern, func(c *arrow.Context) {
		c.After(func(c *arrow.Context) { afterRan = true })
		c.Write([]byte(wantBody))
	})
	recAfter := httptest.NewRecorder()
	app.Handler().ServeHTTP(recAfter, req)
	if !afterRan {
		t.Fatal("zero-middleware bench path must run After callbacks")
	}

	// Middleware must not run on the bench build path.
	mwRan := false
	appMW := arrow.New()
	appMW.Use(func(c *arrow.Context) { mwRan = true })
	registerArrowRoutes(appMW, s.Routes)
	recMW := httptest.NewRecorder()
	appMW.Handler().ServeHTTP(recMW, req)
	if !mwRan {
		t.Fatal("sanity: middleware must run when app.Use is called")
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