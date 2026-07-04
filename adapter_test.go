package arrow_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fun7257/arrow"
)

func stdLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Pre", "1")
		next.ServeHTTP(w, r)
		w.Header().Set("X-Post", "1")
	})
}

func TestAdaptPostRunsAfterHandler(t *testing.T) {
	var order []string

	app := arrow.New()
	app.Use(arrow.Adapt(stdLogger))
	app.Use(func(c *arrow.Context) {
		c.After(func(c *arrow.Context) {
			order = append(order, "native.after")
		})
	})
	app.GET("/", func(c *arrow.Context) {
		order = append(order, "handler")
		c.WriteHeader(http.StatusCreated)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Handler().ServeHTTP(rec, req)

	if rec.Header().Get("X-Pre") != "1" {
		t.Fatal("expected pre header")
	}
	if order[0] != "handler" {
		t.Fatalf("first event = %q, want handler", order[0])
	}
	if rec.Header().Get("X-Post") != "1" {
		t.Fatal("expected post header after handler")
	}
	if len(order) < 2 || order[1] != "native.after" {
		t.Fatalf("order = %v, want handler then native.after", order)
	}
}

func TestLinearMiddleware(t *testing.T) {
	var order []string

	app := arrow.New()
	app.Use(arrow.Linear(
		func(c *arrow.Context) { order = append(order, "pre") },
		func(c *arrow.Context) { order = append(order, "post") },
	))
	app.GET("/", func(c *arrow.Context) { order = append(order, "handler") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Handler().ServeHTTP(rec, req)

	want := []string{"pre", "handler", "post"}
	for i, v := range want {
		if order[i] != v {
			t.Fatalf("order[%d] = %q, want %q (full: %v)", i, order[i], v, order)
		}
	}
}