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
	app.Use(func(c *arrow.Context) {
		called = true
	})

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