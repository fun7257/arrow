package arrow_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fun7257/arrow"
)

func TestRouterMethods(t *testing.T) {
	app := arrow.New()

	app.GET("/get", func(c *arrow.Context) { c.Write([]byte("get")) })
	app.POST("/post", func(c *arrow.Context) { c.Write([]byte("post")) })
	app.PUT("/put", func(c *arrow.Context) { c.Write([]byte("put")) })
	app.DELETE("/delete", func(c *arrow.Context) { c.Write([]byte("delete")) })
	app.PATCH("/patch", func(c *arrow.Context) { c.Write([]byte("patch")) })
	app.HEAD("/head", func(c *arrow.Context) {})
	app.OPTIONS("/options", func(c *arrow.Context) { c.Write([]byte("options")) })

	cases := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodGet, "/get", "get"},
		{http.MethodPost, "/post", "post"},
		{http.MethodPut, "/put", "put"},
		{http.MethodDelete, "/delete", "delete"},
		{http.MethodPatch, "/patch", "patch"},
		{http.MethodOptions, "/options", "options"},
	}

	for _, tc := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(tc.method, tc.path, nil)
		app.Handler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s %s: status = %d", tc.method, tc.path, rec.Code)
		}
		body, _ := io.ReadAll(rec.Body)
		if string(body) != tc.want {
			t.Fatalf("%s %s: body = %q, want %q", tc.method, tc.path, body, tc.want)
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/head", nil)
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("HEAD status = %d", rec.Code)
	}
}

func TestRouterPathValue(t *testing.T) {
	app := arrow.New()
	app.GET("/posts/{id}", func(c *arrow.Context) {
		c.Write([]byte(c.Request.PathValue("id")))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/posts/42", nil)
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if got := rec.Body.String(); got != "42" {
		t.Fatalf("body = %q, want 42", got)
	}
}

func TestGroupPrefix(t *testing.T) {
	app := arrow.New()
	api := app.Group("/api")
	api.GET("/posts", func(c *arrow.Context) { c.Write([]byte("ok")) })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestNestedGroupPrefix(t *testing.T) {
	app := arrow.New()
	v1 := app.Group("/api").Group("/v1")
	v1.GET("/users/{id}", func(c *arrow.Context) {
		c.Write([]byte(c.Request.PathValue("id")))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/alice", nil)
	app.Handler().ServeHTTP(rec, req)

	if got := rec.Body.String(); got != "alice" {
		t.Fatalf("body = %q, want alice", got)
	}
}

func TestGroupMiddlewareInheritance(t *testing.T) {
	var order []string

	app := arrow.New()
	app.Use(func(c *arrow.Context) { order = append(order, "global") })

	api := app.Group("/api")
	api.Use(func(c *arrow.Context) { order = append(order, "api") })
	api.GET("/x", func(c *arrow.Context) { order = append(order, "handler") })

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
	api.Use(func(c *arrow.Context) { apiCalled = true })
	api.GET("/x", func(c *arrow.Context) {})

	admin := app.Group("/admin")
	admin.Use(func(c *arrow.Context) { adminCalled = true })
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

func TestGroupExplicitUseRegistration(t *testing.T) {
	called := false

	app := arrow.New()
	api := app.Group("/api")
	api.Use(func(c *arrow.Context) { called = true })
	api.GET("/x", func(c *arrow.Context) {})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	app.Handler().ServeHTTP(rec, req)

	if !called {
		t.Fatal("group Use middleware should run")
	}
}

func TestZeroMiddlewareDispatchPath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	app := arrow.New()
	app.GET("/ping", func(c *arrow.Context) { c.Write([]byte("pong")) })

	arrow.ResetZeroMiddlewareDispatchCounters()
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || rec.Body.String() != "pong" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}
	if got := arrow.ZeroMiddlewareRouterDispatches(); got != 1 {
		t.Fatalf("router dispatches = %d, want 1", got)
	}
	if got := arrow.ZeroMiddlewarePipelineDispatches(); got != 0 {
		t.Fatalf("pipeline dispatches = %d, want 0", got)
	}

	mwRan := false
	appMW := arrow.New()
	appMW.Use(func(c *arrow.Context) { mwRan = true })
	appMW.GET("/ping", func(c *arrow.Context) { c.Write([]byte("pong")) })

	arrow.ResetZeroMiddlewareDispatchCounters()
	appMW.Handler().ServeHTTP(httptest.NewRecorder(), req)
	if !mwRan {
		t.Fatal("middleware must run when app.Use is called")
	}
	if arrow.ZeroMiddlewareRouterDispatches() != 0 {
		t.Fatalf("middleware router dispatches = %d, want 0", arrow.ZeroMiddlewareRouterDispatches())
	}
}