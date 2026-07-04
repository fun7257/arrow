package arrow_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fun7257/arrow"
)

func TestGroupMiddlewareInheritance(t *testing.T) {
	var order []string

	app := arrow.New()
	app.Use(func(c *arrow.Context) {
		order = append(order, "global")
	})

	api := app.Group("/api")
	api.Use(func(c *arrow.Context) {
		order = append(order, "api")
	})
	api.GET("/x", func(c *arrow.Context) {
		order = append(order, "handler")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	app.Handler().ServeHTTP(rec, req)

	want := []string{"global", "api", "handler"}
	for i, v := range want {
		if order[i] != v {
			t.Fatalf("order[%d] = %q, want %q (full: %v)", i, order[i], v, order)
		}
	}
}

func TestGroupMiddlewareNotInheritedBySibling(t *testing.T) {
	apiCalled := false
	adminCalled := false

	app := arrow.New()
	api := app.Group("/api")
	api.Use(func(c *arrow.Context) {
		apiCalled = true
	})
	api.GET("/x", func(c *arrow.Context) {})

	admin := app.Group("/admin")
	admin.Use(func(c *arrow.Context) {
		adminCalled = true
	})
	admin.GET("/x", func(c *arrow.Context) {})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/x", nil)
	app.Handler().ServeHTTP(rec, req)

	if apiCalled {
		t.Fatal("api middleware should not run on admin routes")
	}
	if !adminCalled {
		t.Fatal("admin middleware should run on admin routes")
	}
}

func TestGroupUseChaining(t *testing.T) {
	called := false

	app := arrow.New()
	api := app.Group("/api").Use(func(c *arrow.Context) {
		called = true
	})
	api.GET("/x", func(c *arrow.Context) {})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	app.Handler().ServeHTTP(rec, req)

	if !called {
		t.Fatal("chained Use middleware should run")
	}
}