// Package arrow_test provides stdlib parity helpers.
//
// Scope: Arrow delegates routing to http.ServeMux, wraps http.ResponseWriter,
// and exposes http.Handler mounting. Parity tests compare observable HTTP
// outcomes (status, body, Allow, PathValue) between Arrow and a baseline mux.
// Out of scope: net/http Client, Transport, middleware execution semantics.
package arrow_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// parityResult captures observable HTTP response fields for comparison.
type parityResult struct {
	Status    int
	Body      string
	Allow     string
	PathValue string
}

func serveAndCapture(h http.Handler, req *http.Request) parityResult {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return parityResult{
		Status: rec.Code,
		Body:   rec.Body.String(),
		Allow:  rec.Header().Get("Allow"),
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
		io.WriteString(w, prefix+r.PathValue(key))
	})
}