package arrow

import (
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestRouterZeroMiddlewareUsesRunNoMiddleware(t *testing.T) {
	var calls atomic.Uint32
	prev := hookRunNoMiddleware
	hookRunNoMiddleware = func() { calls.Add(1) }
	defer func() { hookRunNoMiddleware = prev }()

	app := New()
	app.GET("/ping", func(c *Context) { c.Write([]byte("pong")) })

	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/ping", nil))

	if calls.Load() != 1 {
		t.Fatalf("runNoMiddleware invocations = %d, want 1", calls.Load())
	}
	if rec.Body.String() != "pong" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "pong")
	}
}