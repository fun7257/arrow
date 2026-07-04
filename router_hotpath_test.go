package arrow_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fun7257/arrow"
)

// Zero-middleware routes use executeZeroMiddleware via router (see router_dispatch_test.go).
// These tests drive app.Handler().ServeHTTP without app.Use — the same path
// exercised by minimal/static benchmarks.

func TestZeroMiddlewareAfterFromHandler(t *testing.T) {
	var order []string

	app := arrow.New()
	app.GET("/", func(c *arrow.Context) {
		c.After(func(c *arrow.Context) { order = append(order, "after") })
		order = append(order, "handler")
	})

	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	want := []string{"handler", "after"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i, v := range want {
		if order[i] != v {
			t.Fatalf("order[%d] = %q, want %q (full: %v)", i, order[i], v, order)
		}
	}
}

func TestZeroMiddlewareAbortInHandler(t *testing.T) {
	var afterRan bool

	app := arrow.New()
	app.GET("/", func(c *arrow.Context) {
		c.After(func(c *arrow.Context) { afterRan = true })
		c.Abort(http.StatusTeapot)
	})

	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
	if !afterRan {
		t.Fatal("After callback should still run after Abort in handler")
	}
}

func TestZeroMiddlewarePanicRecovery(t *testing.T) {
	app := arrow.New()
	app.GET("/", func(c *arrow.Context) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}