package arrow_test

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

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

func TestMiddlewareRunsOnlyWhenUsed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	app := arrow.New()
	app.GET("/ping", func(c *arrow.Context) { c.Write([]byte("pong")) })
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "pong" {
		t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
	}

	mwRan := false
	appMW := arrow.New()
	appMW.Use(func(c *arrow.Context) { mwRan = true })
	appMW.GET("/ping", func(c *arrow.Context) { c.Write([]byte("pong")) })
	appMW.Handler().ServeHTTP(httptest.NewRecorder(), req)
	if !mwRan {
		t.Fatal("middleware must run when app.Use is called")
	}
}

func TestAnyRoute(t *testing.T) {
	app := arrow.New()
	app.Any("/any", func(c *arrow.Context) {
		c.Write([]byte(c.Request.Method))
	})

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(method, "/any", nil)
		app.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status=%d", method, rec.Code)
		}
		if rec.Body.String() != method {
			t.Fatalf("%s: body=%q", method, rec.Body.String())
		}
	}
}

func TestHandleHTTPFileServer(t *testing.T) {
	static := fstest.MapFS{
		"hello.txt": &fstest.MapFile{Data: []byte("hello")},
	}

	app := arrow.New()
	app.HandleHTTP("/static/", http.FileServer(http.FS(static)))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/static/hello.txt", nil)
	app.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "hello" {
		t.Fatalf("body=%q", body)
	}
}

func TestHandleHTTPWithMiddleware(t *testing.T) {
	called := false
	app := arrow.New()
	app.Use(func(c *arrow.Context) { called = true })

	static := fstest.MapFS{"a.txt": &fstest.MapFile{Data: []byte("a")}}
	app.HandleHTTP("/files/", http.FileServer(http.FS(static)))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/files/a.txt", nil)
	app.Handler().ServeHTTP(rec, req)

	if !called {
		t.Fatal("middleware should run for HandleHTTP routes")
	}
	if rec.Body.String() != "a" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestMuxExpose(t *testing.T) {
	app := arrow.New()
	mux := app.Mux()
	if mux == nil {
		t.Fatal("Mux() returned nil")
	}

	direct := false
	mux.HandleFunc("GET /direct", func(w http.ResponseWriter, r *http.Request) {
		direct = true
	})

	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/direct", nil))
	if !direct {
		t.Fatal("direct mux registration should work")
	}
}

func TestHandleHTTPMethod(t *testing.T) {
	app := arrow.New()
	app.HandleHTTPMethod(http.MethodGet, "/only-get", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/only-get", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET: %d", rec.Code)
	}

	rec2 := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec2, httptest.NewRequest(http.MethodPost, "/only-get", nil))
	if rec2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST: %d, want 405", rec2.Code)
	}
}

func TestGroupHandleHTTP(t *testing.T) {
	static := fstest.MapFS{"x.txt": &fstest.MapFile{Data: []byte("x")}}
	app := arrow.New()
	api := app.Group("/api")
	api.HandleHTTP("/static/", http.FileServer(http.FS(static)))

	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/static/x.txt", nil))
	if rec.Body.String() != "x" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

var _ fs.FS = fstest.MapFS{}