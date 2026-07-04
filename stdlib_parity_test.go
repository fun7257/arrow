// Package arrow_test provides stdlib parity helpers.
//
// Scope: Arrow delegates routing to http.ServeMux, wraps http.ResponseWriter,
// and exposes http.Handler mounting. Parity tests compare observable HTTP
// outcomes (status, body, Allow, PathValue) and ResponseWriter optional
// interface sets between Arrow and a baseline mux.
// Out of scope: net/http Client, Transport, middleware execution semantics.
package arrow_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const parityPathValueHeader = "X-Parity-PathValue"

// parityResult captures observable HTTP response fields for comparison.
type parityResult struct {
	Status    int
	Body      string
	Allow     string
	PathValue string
}

// rwInterfaces records which optional ResponseWriter interfaces are supported.
type rwInterfaces struct {
	Flusher    bool
	Hijacker   bool
	Pusher     bool
	ReaderFrom bool
}

func probeRW(w http.ResponseWriter) rwInterfaces {
	_, f := w.(http.Flusher)
	_, h := w.(http.Hijacker)
	_, p := w.(http.Pusher)
	_, r := w.(io.ReaderFrom)
	return rwInterfaces{Flusher: f, Hijacker: h, Pusher: p, ReaderFrom: r}
}

func (i rwInterfaces) String() string {
	return fmt.Sprintf("F=%v,H=%v,P=%v,R=%v", i.Flusher, i.Hijacker, i.Pusher, i.ReaderFrom)
}

func assertRWParity(t *testing.T, name string, baseline, subject rwInterfaces) {
	t.Helper()
	if baseline != subject {
		t.Errorf("%s: interface set baseline=%s subject=%s", name, baseline, subject)
	}
}

func serveAndCapture(h http.Handler, req *http.Request) parityResult {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return parityResult{
		Status:    rec.Code,
		Body:      rec.Body.String(),
		Allow:     rec.Header().Get("Allow"),
		PathValue: rec.Header().Get(parityPathValueHeader),
	}
}

func assertParity(t *testing.T, name string, baseline, subject http.Handler, req *http.Request) {
	t.Helper()
	b := serveAndCapture(baseline, req)
	s := serveAndCapture(subject, req)

	if b.Status != s.Status {
		t.Errorf("%s: status baseline=%d subject=%d", name, b.Status, s.Status)
	}
	if b.Body != s.Body {
		t.Errorf("%s: body baseline=%q subject=%q", name, b.Body, s.Body)
	}
	if b.Allow != s.Allow {
		t.Errorf("%s: Allow baseline=%q subject=%q", name, b.Allow, s.Allow)
	}
	if b.PathValue != s.PathValue {
		t.Errorf("%s: PathValue baseline=%q subject=%q", name, b.PathValue, s.PathValue)
	}
}

func newRequest(method, path, host string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	if host != "" {
		req.Host = host
	}
	return req
}

// expectedMuxPattern mirrors arrow.Router mux pattern construction for parity baselines.
func expectedMuxPattern(method, pattern string, handleHTTPSubtree bool) string {
	fullPattern := pattern
	if handleHTTPSubtree && strings.HasSuffix(fullPattern, "/") && !strings.Contains(fullPattern, "{") {
		dir := strings.TrimSuffix(fullPattern, "/")
		fullPattern = dir + "/{path...}"
	}
	if method == "" {
		return fullPattern
	}
	return method + " " + fullPattern
}

func registerBaselineHandler(mux *http.ServeMux, method, pattern string, h http.Handler) {
	mux.Handle(expectedMuxPattern(method, pattern, false), h)
}

func registerBaselineHTTP(mux *http.ServeMux, method, pattern string, h http.Handler) {
	fullPattern := pattern
	handler := h
	muxPattern := expectedMuxPattern(method, pattern, false)

	if strings.HasSuffix(fullPattern, "/") && !strings.Contains(fullPattern, "{") {
		handler = http.StripPrefix(fullPattern, h)
		muxPattern = expectedMuxPattern(method, pattern, true)
	}
	mux.Handle(muxPattern, handler)
}

func stdHandler(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, body)
	})
}

func stdHandlerMethod(body string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, body+":"+r.Method)
	})
}

func stdHandlerPathValue(key, prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pv := r.PathValue(key)
		w.Header().Set(parityPathValueHeader, pv)
		io.WriteString(w, prefix+pv)
	})
}