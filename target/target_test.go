package target_test

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/fun7257/arrow"
	"github.com/fun7257/arrow/target"
)

func runHandler(t *testing.T, method, path string, handler arrow.HandlerFunc, setup func(*http.Request)) (*httptest.ResponseRecorder, *arrow.Context) {
	t.Helper()
	var captured *arrow.Context
	app := arrow.New()
	app.Handle(method, "/", func(c *arrow.Context) {
		captured = c
		handler(c)
	})
	req := httptest.NewRequest(method, path, nil)
	if setup != nil {
		setup(req)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	if captured == nil {
		t.Fatal("handler was not invoked")
	}
	return rec, captured
}

func TestWriteJSONTypes(t *testing.T) {
	t.Run("struct", func(t *testing.T) {
		rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
			type payload struct {
				Name string `json:"name"`
			}
			if err := target.WriteJSON(c, http.StatusOK, payload{Name: "arrow"}); err != nil {
				t.Fatal(err)
			}
		}, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("content-type = %q", ct)
		}
		if body := strings.TrimSpace(rec.Body.String()); body != `{"name":"arrow"}` {
			t.Fatalf("body = %q", body)
		}
	})

	t.Run("slice", func(t *testing.T) {
		rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
			if err := target.WriteJSON(c, http.StatusOK, []int{1, 2, 3}); err != nil {
				t.Fatal(err)
			}
		}, nil)
		if body := strings.TrimSpace(rec.Body.String()); body != `[1,2,3]` {
			t.Fatalf("body = %q", body)
		}
	})

	t.Run("map", func(t *testing.T) {
		rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
			if err := target.WriteJSON(c, http.StatusOK, map[string]int{"a": 1}); err != nil {
				t.Fatal(err)
			}
		}, nil)
		if !strings.Contains(rec.Body.String(), `"a":1`) {
			t.Fatalf("body = %q", rec.Body.String())
		}
	})
}

func TestWriteJSONAsEnvelope(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		type user struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		err := target.WriteJSONAs(c, http.StatusOK, user{ID: 1, Name: "ada"}, func(u user) target.Envelope[user] {
			return target.Envelope[user]{
				Code:    0,
				Message: "ok",
				Data:    u,
			}
		})
		if err != nil {
			t.Fatal(err)
		}
	}, nil)

	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	var got target.Envelope[user]
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Code != 0 || got.Message != "ok" || got.Data.ID != 1 || got.Data.Name != "ada" {
		t.Fatalf("envelope = %+v", got)
	}
}

func TestOKAs(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		if err := target.OKAs(c, "raw", func(s string) target.Envelope[string] {
			return target.Envelope[string]{Code: 0, Message: "ok", Data: s}
		}); err != nil {
			t.Fatal(err)
		}
	}, nil)
	if !strings.Contains(rec.Body.String(), `"data":"raw"`) {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestWriteEncoded(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		err := target.WriteEncoded(c, target.Encoded[map[string]string]{
			Status:  http.StatusOK,
			Encoder: target.JSONEncoder[map[string]string]{},
			Body:    map[string]string{"k": "v"},
		})
		if err != nil {
			t.Fatal(err)
		}
	}, nil)
	if !strings.Contains(rec.Body.String(), `"k":"v"`) {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestErrorShape(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		if err := target.WriteError(c, http.StatusBadRequest, "invalid"); err != nil {
			t.Fatal(err)
		}
	}, nil)
	var got target.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Message != "invalid" {
		t.Fatalf("error = %+v", got)
	}
}

func TestProblemShape(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		p := target.Problem{
			Type:   "about:blank",
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: "missing",
			Extra:  map[string]string{"trace_id": "abc"},
		}
		if err := target.WriteProblem(c, p); err != nil {
			t.Fatal(err)
		}
	}, nil)
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["title"] != "Not Found" || int(got["status"].(float64)) != http.StatusNotFound {
		t.Fatalf("problem = %+v", got)
	}
	if got["trace_id"] != "abc" {
		t.Fatalf("extra = %+v", got)
	}
}

func TestPageShape(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		page := target.Page[string]{
			Items: []string{"a", "b"},
			Total: 2,
			Page:  1,
			Size:  10,
		}
		if err := target.WritePage(c, http.StatusOK, page); err != nil {
			t.Fatal(err)
		}
	}, nil)
	var got target.Page[string]
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Total != 2 || len(got.Items) != 2 || got.Items[0] != "a" {
		t.Fatalf("page = %+v", got)
	}
}

func TestAbortPenetration(t *testing.T) {
	handlerCalled := false

	app := arrow.New()
	app.Use(func(c *arrow.Context) {
		_ = target.AbortUnauthorized(c, "unauthorized")
	})
	app.GET("/", func(c *arrow.Context) {
		handlerCalled = true
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Handler().ServeHTTP(rec, req)

	if handlerCalled {
		t.Fatal("handler should not run after AbortUnauthorized")
	}
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	var got target.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Message != "unauthorized" {
		t.Fatalf("error = %+v", got)
	}
}

func TestAbortStillRunsAfter(t *testing.T) {
	var order []string

	app := arrow.New()
	app.Use(func(c *arrow.Context) {
		c.After(func(c *arrow.Context) { order = append(order, "after") })
		_ = target.AbortForbidden(c, "denied")
	})
	app.GET("/", func(c *arrow.Context) { order = append(order, "handler") })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Handler().ServeHTTP(rec, req)

	if len(order) != 1 || order[0] != "after" {
		t.Fatalf("order = %v, want [after]", order)
	}
}

func TestAbortWithDelegatesToAbort(t *testing.T) {
	handlerCalled := false
	app := arrow.New()
	app.Use(func(c *arrow.Context) {
		_ = target.AbortWith(c, target.JSON(http.StatusTeapot, target.Error{Message: "short"}))
	})
	app.GET("/", func(c *arrow.Context) { handlerCalled = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Handler().ServeHTTP(rec, req)

	if handlerCalled {
		t.Fatal("handler should not run")
	}
	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
}

func TestAbortProblem(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		_ = target.AbortProblem(c, target.Problem{
			Title:  "Bad",
			Status: http.StatusBadRequest,
			Detail: "nope",
		})
	}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestNoDoubleWrite(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		if err := target.WriteJSON(c, http.StatusOK, map[string]string{"first": "yes"}); err != nil {
			t.Fatal(err)
		}
		if !c.Written() {
			t.Fatal("expected context to be written after first write")
		}
		if err := target.WriteJSON(c, http.StatusOK, map[string]string{"second": "no"}); err != nil {
			t.Fatal(err)
		}
	}, nil)
	body := rec.Body.String()
	if !strings.Contains(body, `"first":"yes"`) {
		t.Fatalf("body = %q", body)
	}
	if strings.Contains(body, `"second"`) {
		t.Fatalf("second write should be ignored: %q", body)
	}
}

func TestNoDoubleWriteAfterRedirect(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		if err := target.WriteRedirect(c, http.StatusFound, "/next"); err != nil {
			t.Fatal(err)
		}
		if !c.Written() {
			t.Fatal("expected context to be written after redirect")
		}
		_ = target.WriteJSON(c, http.StatusOK, map[string]string{"second": "no"})
	}, nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
}

func TestWriteRedirect(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		if err := target.Found(c, "/login"); err != nil {
			t.Fatal(err)
		}
		if !c.Written() {
			t.Fatal("expected written after redirect")
		}
	}, nil)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("location = %q", loc)
	}
}

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello file"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		if err := target.WriteFile(c, path); err != nil {
			t.Fatal(err)
		}
		if !c.Written() {
			t.Fatal("expected written after file serve")
		}
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Body.String() != "hello file" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestNoDoubleWriteAfterFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		if err := target.WriteFile(c, path); err != nil {
			t.Fatal(err)
		}
		if !c.Written() {
			t.Fatal("expected written after file serve")
		}
		_ = target.WriteJSON(c, http.StatusOK, map[string]string{"ignored": "yes"})
	}, nil)
	if rec.Body.String() != "hello" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestWriteFileFS(t *testing.T) {
	fsys := fstest.MapFS{
		"data.txt": &fstest.MapFile{Data: []byte("fs-data")},
	}
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		if err := target.WriteFileFS(c, fsys, "data.txt"); err != nil {
			t.Fatal(err)
		}
	}, nil)
	if rec.Body.String() != "fs-data" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestWriteSSE(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		err := target.WriteSSE(c, func(w *target.EventWriter) error {
			return w.Data("hello")
		})
		if err != nil {
			t.Fatal(err)
		}
		if !c.Written() {
			t.Fatal("expected written after sse")
		}
	}, nil)
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "data: hello") {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestWriteNegotiated(t *testing.T) {
	t.Run("json default", func(t *testing.T) {
		rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
			_ = target.WriteNegotiated(c, http.StatusOK, map[string]string{"fmt": "json"})
		}, nil)
		if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
			t.Fatalf("content-type = %q", ct)
		}
	})

	t.Run("xml accept", func(t *testing.T) {
		rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
			_ = target.WriteNegotiated(c, http.StatusOK, struct {
				XMLName struct{} `xml:"payload"`
				Fmt     string   `xml:"fmt"`
			}{Fmt: "xml"})
		}, func(r *http.Request) {
			r.Header.Set("Accept", "application/xml;q=1,text/html;q=0.9")
		})
		if ct := rec.Header().Get("Content-Type"); ct != "application/xml; charset=utf-8" {
			t.Fatalf("content-type = %q", ct)
		}
		if !strings.Contains(rec.Body.String(), "<payload>") {
			t.Fatalf("body = %q", rec.Body.String())
		}
	})
}

func TestEncodeErrorNoHeaders(t *testing.T) {
	var written bool
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		_ = target.WriteJSON(c, http.StatusOK, func() {})
		written = c.Written()
	}, nil)
	if written {
		t.Fatal("expected no response committed on encode failure")
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty on encode failure", rec.Body.String())
	}
}



func TestBeforeWriteHook(t *testing.T) {
	target.Default.BeforeWrite = func(c *arrow.Context, t target.Target) (target.Target, error) {
		return target.JSON(http.StatusAccepted, map[string]string{"wrapped": "yes"}), nil
	}
	t.Cleanup(func() { target.Default.BeforeWrite = nil })

	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		_ = target.WriteJSON(c, http.StatusOK, map[string]string{"ignored": "yes"})
	}, nil)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusAccepted)
	}
}

func TestWriteNegotiatedRegisteredEncoder(t *testing.T) {
	const contentType = "application/x-custom"
	target.RegisterEncoder(contentType, func(w io.Writer, v []byte) error {
		_, err := w.Write(append([]byte("custom:"), v...))
		return err
	})
	t.Cleanup(func() { target.RegisterEncoder(contentType, nil) })

	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		_ = target.WriteNegotiated(c, http.StatusOK, map[string]string{"k": "v"})
	}, func(r *http.Request) {
		r.Header.Set("Accept", contentType)
	})
	if ct := rec.Header().Get("Content-Type"); ct != contentType {
		t.Fatalf("content-type = %q", ct)
	}
	if !strings.HasPrefix(rec.Body.String(), "custom:") {
		t.Fatalf("body = %q, want custom encoder prefix", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"k":"v"`) {
		t.Fatalf("body = %q, want JSON payload inside custom encoder output", rec.Body.String())
	}
}

func TestTemplateExecuteFailure(t *testing.T) {
	var written bool
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		tmpl := template.Must(template.New("bad").Parse(`{{.Age}}`))
		_ = target.WriteTemplate(c, http.StatusOK, tmpl, struct{ Name string }{Name: "ada"})
		written = c.Written()
	}, nil)
	if written {
		t.Fatal("expected no response committed on template failure")
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty on template failure", rec.Body.String())
	}
}

func TestOnEncodeErrorHook(t *testing.T) {
	var hookCalled bool
	var hookErr error
	target.Default.OnEncodeError = func(c *arrow.Context, err error) {
		hookCalled = true
		hookErr = err
	}
	t.Cleanup(func() { target.Default.OnEncodeError = nil })

	_, _ = runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		_ = target.WriteJSON(c, http.StatusOK, func() {})
	}, nil)
	if !hookCalled {
		t.Fatal("expected OnEncodeError hook to be called")
	}
	if hookErr == nil {
		t.Fatal("expected hook to receive encode error")
	}
}

func TestAbortEarlyReturnWhenWritten(t *testing.T) {
	handlerCalled := false
	var aborted bool
	var status int

	app := arrow.New()
	app.Use(func(c *arrow.Context) {
		_ = target.WriteJSON(c, http.StatusOK, map[string]string{"first": "yes"})
		_ = target.Abort(c, target.JSON(http.StatusTeapot, target.Error{Message: "ignored"}))
		aborted = c.IsAborted()
		status = c.Status()
	})
	app.GET("/", func(c *arrow.Context) { handlerCalled = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	app.Handler().ServeHTTP(rec, req)

	if handlerCalled {
		t.Fatal("handler should not run after abort")
	}
	if !aborted {
		t.Fatal("expected abort after written response")
	}
	if status != http.StatusTeapot {
		t.Fatalf("abort status = %d, want %d", status, http.StatusTeapot)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("committed status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"first":"yes"`) {
		t.Fatalf("body = %q, want first write preserved", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "ignored") {
		t.Fatalf("body = %q, second abort write should be skipped", rec.Body.String())
	}
}

func TestWriteStream(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		err := target.WriteStream(c, http.StatusOK, "text/plain; charset=utf-8", func(w io.Writer) error {
			_, err := io.WriteString(w, "streamed")
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if rec.Body.String() != "streamed" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestWriteStreamReader(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		err := target.WriteStreamReader(c, http.StatusOK, "text/plain; charset=utf-8", strings.NewReader("reader"))
		if err != nil {
			t.Fatal(err)
		}
	}, nil)
	if rec.Body.String() != "reader" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

func TestProblemExtraSkipsReservedKeys(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		_ = target.WriteProblem(c, target.Problem{
			Title: "Conflict",
			Extra: map[string]string{
				"title":    "overwritten",
				"trace_id": "abc",
			},
		})
	}, nil)
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["title"] != "Conflict" {
		t.Fatalf("title = %v, want standard field preserved", got["title"])
	}
	if got["trace_id"] != "abc" {
		t.Fatalf("trace_id = %v, want extension field", got["trace_id"])
	}
}

func TestWriteNegotiatedStarAccept(t *testing.T) {
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		_ = target.WriteNegotiated(c, http.StatusOK, map[string]string{"fmt": "json"})
	}, func(r *http.Request) {
		r.Header.Set("Accept", "*/*")
	})
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
}

func TestRegisterEncoderNilClears(t *testing.T) {
	const contentType = "application/x-clear-test"
	target.RegisterEncoder(contentType, func(w io.Writer, v []byte) error {
		return errors.New("should not run")
	})
	target.RegisterEncoder(contentType, nil)
	rec, _ := runHandler(t, http.MethodGet, "/", func(c *arrow.Context) {
		_ = target.WriteBytes(c, http.StatusOK, contentType, []byte("raw"))
	}, nil)
	if rec.Body.String() != "raw" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

