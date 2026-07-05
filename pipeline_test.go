package arrow_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fun7257/arrow"
	"github.com/fun7257/arrow/middleware"
)

func TestPipelineAfterForwardOrder(t *testing.T) {
	var order []string

	app := arrow.New()
	app.Use(func(c *arrow.Context) {
		c.After(func(c *arrow.Context) { order = append(order, "m1.after") })
	})
	app.Use(func(c *arrow.Context) {
		c.After(func(c *arrow.Context) { order = append(order, "m2.after") })
	})
	app.GET("/", func(c *arrow.Context) {
		order = append(order, "handler")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Handler().ServeHTTP(rec, req)

	want := []string{"handler", "m1.after", "m2.after"}
	if len(order) != len(want) {
		t.Fatalf("order len = %d, want %d (%v)", len(order), len(want), order)
	}
	for i, v := range want {
		if order[i] != v {
			t.Fatalf("order[%d] = %q, want %q (full: %v)", i, order[i], v, order)
		}
	}
}

func TestPipelinePreForwardOrder(t *testing.T) {
	var order []string

	app := arrow.New()
	app.Use(func(c *arrow.Context) { order = append(order, "m1.pre") })
	app.Use(func(c *arrow.Context) { order = append(order, "m2.pre") })
	app.GET("/", func(c *arrow.Context) { order = append(order, "handler") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Handler().ServeHTTP(rec, req)

	want := []string{"m1.pre", "m2.pre", "handler"}
	for i, v := range want {
		if order[i] != v {
			t.Fatalf("order[%d] = %q, want %q (full: %v)", i, order[i], v, order)
		}
	}
}

func TestPipelineAbortSkipsHandler(t *testing.T) {
	handlerCalled := false

	app := arrow.New()
	app.Use(func(c *arrow.Context) {
		c.Abort(http.StatusUnauthorized)
	})
	app.GET("/", func(c *arrow.Context) {
		handlerCalled = true
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Handler().ServeHTTP(rec, req)

	if handlerCalled {
		t.Fatal("handler should not run after abort")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestPipelineAbortStillRunsAfter(t *testing.T) {
	var order []string

	app := arrow.New()
	app.Use(func(c *arrow.Context) {
		c.After(func(c *arrow.Context) { order = append(order, "m1.after") })
		c.Abort(http.StatusForbidden)
	})
	app.Use(func(c *arrow.Context) {
		order = append(order, "m2.pre")
	})
	app.GET("/", func(c *arrow.Context) { order = append(order, "handler") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Handler().ServeHTTP(rec, req)

	want := []string{"m1.after"}
	for i, v := range want {
		if order[i] != v {
			t.Fatalf("order[%d] = %q, want %q (full: %v)", i, order[i], v, order)
		}
	}
}

func TestPipelinePanicRecovery(t *testing.T) {
	app := arrow.New()
	app.Use(middleware.Recover())
	app.GET("/", func(c *arrow.Context) {
		panic("boom")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}