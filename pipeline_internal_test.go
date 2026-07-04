package arrow

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRunNoMiddlewareAfterOrder(t *testing.T) {
	var order []string

	rec := httptest.NewRecorder()
	ctx := newContext(rec, httptest.NewRequest("GET", "/", nil))

	runNoMiddleware(ctx, func(c *Context) {
		c.After(func(c *Context) { order = append(order, "after") })
		order = append(order, "handler")
	})

	want := []string{"handler", "after"}
	if len(order) != len(want) {
		t.Fatalf("order = %v, want %v", order, want)
	}
	for i, v := range want {
		if order[i] != v {
			t.Fatalf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestServeZeroMiddlewareFromHTTPEquivalentToRunNoMiddleware(t *testing.T) {
	runCase := func(handler HandlerFunc, viaRouter bool) (code int, body string, handlerRan bool) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		var ran bool
		wrapped := func(c *Context) {
			ran = true
			handler(c)
		}
		resetZeroMiddlewareDispatchCounters()
		if viaRouter {
			serveZeroMiddlewareFromHTTP(rec, req, wrapped)
		} else {
			ctx := newContext(rec, req)
			runNoMiddleware(ctx, wrapped)
		}
		return rec.Code, rec.Body.String(), ran
	}

	t.Run("handler and after", func(t *testing.T) {
		handler := func(c *Context) {
			c.After(func(c *Context) { c.Write([]byte("-after")) })
			c.Write([]byte("handler"))
		}
		rCode, rBody, rRan := runCase(handler, true)
		pCode, pBody, pRan := runCase(handler, false)
		if rCode != pCode || rBody != pBody || !rRan || !pRan {
			t.Fatalf("router (%d,%q) != pipeline (%d,%q)", rCode, rBody, pCode, pBody)
		}
	})

	t.Run("abort in handler", func(t *testing.T) {
		var afterRan bool
		handler := func(c *Context) {
			c.After(func(c *Context) { afterRan = true })
			c.Abort(http.StatusTeapot)
		}
		rCode, _, _ := runCase(handler, true)
		pCode, _, _ := runCase(handler, false)
		if rCode != http.StatusTeapot || pCode != http.StatusTeapot {
			t.Fatalf("status router=%d pipeline=%d", rCode, pCode)
		}
		if !afterRan {
			t.Fatal("After must run after Abort on both paths")
		}
	})

	t.Run("panic recovery", func(t *testing.T) {
		handler := func(c *Context) { panic("boom") }
		rCode, _, _ := runCase(handler, true)
		pCode, _, _ := runCase(handler, false)
		if rCode != http.StatusInternalServerError || pCode != http.StatusInternalServerError {
			t.Fatalf("panic status router=%d pipeline=%d", rCode, pCode)
		}
	})
}

func TestExecuteZeroMiddlewareSkipsHandlerWhenPreAborted(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx := newContext(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	ctx.aborted = true
	ctx.code = http.StatusTeapot
	ctx.sw.WriteHeader(http.StatusTeapot)

	handlerRan := false
	executeZeroMiddleware(ctx, func(c *Context) { handlerRan = true })

	if handlerRan {
		t.Fatal("executeZeroMiddleware must skip handler when already aborted")
	}
}

func TestRunNoMiddlewareIncrementsPipelineCounter(t *testing.T) {
	resetZeroMiddlewareDispatchCounters()
	rec := httptest.NewRecorder()
	ctx := newContext(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	runNoMiddleware(ctx, func(c *Context) {})
	if zeroMiddlewarePipelineDispatches.Load() != 1 {
		t.Fatalf("pipeline dispatches = %d, want 1", zeroMiddlewarePipelineDispatches.Load())
	}
	if zeroMiddlewareRouterDispatches.Load() != 0 {
		t.Fatalf("router dispatches = %d, want 0", zeroMiddlewareRouterDispatches.Load())
	}
}