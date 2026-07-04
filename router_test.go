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
		id := c.Request.PathValue("id")
		c.Write([]byte(id))
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/posts/42", nil)
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "42" {
		t.Fatalf("body = %q, want %q", body, "42")
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

	body, _ := io.ReadAll(rec.Body)
	if string(body) != "alice" {
		t.Fatalf("body = %q, want alice", body)
	}
}