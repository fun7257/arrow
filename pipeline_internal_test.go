package arrow

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// simulateInlineZeroMiddleware mirrors the router zero-middleware inline closure.
func simulateInlineZeroMiddleware(ctx *Context, handler HandlerFunc) {
	defer recoverAndRelease(ctx)
	handler(ctx)
	for _, after := range ctx.afters {
		after(ctx)
	}
}

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

func TestZeroMiddlewareInlineEquivalentToRunNoMiddleware(t *testing.T) {
	t.Helper()

	type outcome struct {
		code       int
		body       string
		order      []string
		afterRan   bool
		handlerRan bool
	}

	runCase := func(label string, setup func(*Context), handler HandlerFunc, viaInline bool) outcome {
		rec := httptest.NewRecorder()
		ctx := newContext(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if setup != nil {
			setup(ctx)
		}
		var o outcome
		wrapped := func(c *Context) {
			o.handlerRan = true
			handler(c)
		}
		if viaInline {
			simulateInlineZeroMiddleware(ctx, wrapped)
		} else {
			runNoMiddleware(ctx, wrapped)
		}
		o.code = rec.Code
		o.body = rec.Body.String()
		return o
	}

	t.Run("handler and after", func(t *testing.T) {
		handler := func(c *Context) {
			c.After(func(c *Context) { c.Write([]byte("-after")) })
			c.Write([]byte("handler"))
		}
		inline := runCase("inline", nil, handler, true)
		viaRun := runCase("runNoMiddleware", nil, handler, false)
		if inline.code != viaRun.code || inline.body != viaRun.body {
			t.Fatalf("inline (%d,%q) != runNoMiddleware (%d,%q)", inline.code, inline.body, viaRun.code, viaRun.body)
		}
		if !inline.handlerRan || !viaRun.handlerRan {
			t.Fatal("handler must run on both paths")
		}
	})

	t.Run("abort in handler", func(t *testing.T) {
		var afterRan bool
		handler := func(c *Context) {
			c.After(func(c *Context) { afterRan = true })
			c.Abort(http.StatusTeapot)
		}
		inline := runCase("inline", nil, handler, true)
		viaRun := runCase("runNoMiddleware", nil, handler, false)
		if inline.code != http.StatusTeapot || viaRun.code != http.StatusTeapot {
			t.Fatalf("status inline=%d run=%d", inline.code, viaRun.code)
		}
		if !afterRan {
			t.Fatal("After must run after Abort on both paths")
		}
	})

	t.Run("panic recovery", func(t *testing.T) {
		handler := func(c *Context) { panic("boom") }
		inline := runCase("inline", nil, handler, true)
		viaRun := runCase("runNoMiddleware", nil, handler, false)
		if inline.code != http.StatusInternalServerError || viaRun.code != http.StatusInternalServerError {
			t.Fatalf("panic status inline=%d run=%d", inline.code, viaRun.code)
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