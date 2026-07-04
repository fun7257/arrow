// HTTP performance benchmarks for Arrow vs net/http.ServeMux baseline.
//
// Scenarios (see testdata/bench/):
//   - minimal:    single route, smallest response
//   - static:     multi-route static table
//   - parametric: {param} routes with PathValue
//   - middleware: static table + 5-layer noop middleware stack
//   - large:      120-route GitHub REST–style table
//
// Run: go test -bench=. -benchmem -count=1 -run='^$' ./...
// Corpus/probe alignment: TestBenchCorpusLoads, TestBenchProbeRequestsAlignWithCorpus.
// Hot path: TestBenchHotPathUsesRouterZeroMiddlewareDispatch (router_dispatch_test.go).
package arrow_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func runBenchServeHTTP(b *testing.B, h http.Handler, req *http.Request) {
	b.Helper()
	b.ReportAllocs()
	rec := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		rec.Body.Reset()
		rec.Code = 0
		h.ServeHTTP(rec, req)
	}
}

// --- Minimal: single route ---

func BenchmarkArrow_Minimal(b *testing.B) {
	s := loadBenchScenario(b, "minimal.json")
	h := buildArrowApp(s)
	req := benchRequest(probeRequest(s))
	b.ResetTimer()
	runBenchServeHTTP(b, h, req)
}

func BenchmarkStdlib_Minimal(b *testing.B) {
	s := loadBenchScenario(b, "minimal.json")
	h := buildStdlibMux(b, s)
	req := benchRequest(probeRequest(s))
	b.ResetTimer()
	runBenchServeHTTP(b, h, req)
}

// --- Static: multi-route table ---

func BenchmarkArrow_Static(b *testing.B) {
	s := loadBenchScenario(b, "static.json")
	h := buildArrowApp(s)
	req := benchRequest(probeRequest(s))
	b.ResetTimer()
	runBenchServeHTTP(b, h, req)
}

func BenchmarkStdlib_Static(b *testing.B) {
	s := loadBenchScenario(b, "static.json")
	h := buildStdlibMux(b, s)
	req := benchRequest(probeRequest(s))
	b.ResetTimer()
	runBenchServeHTTP(b, h, req)
}

// --- Parametric: {param} routes ---

func BenchmarkArrow_Parametric(b *testing.B) {
	s := loadBenchScenario(b, "parametric.json")
	h := buildArrowApp(s)
	req := benchRequest(probeRequest(s))
	b.ResetTimer()
	runBenchServeHTTP(b, h, req)
}

func BenchmarkStdlib_Parametric(b *testing.B) {
	s := loadBenchScenario(b, "parametric.json")
	h := buildStdlibMux(b, s)
	req := benchRequest(probeRequest(s))
	b.ResetTimer()
	runBenchServeHTTP(b, h, req)
}

// --- Middleware: 5-layer noop stack on static table ---

func BenchmarkArrow_Middleware(b *testing.B) {
	s := loadBenchScenario(b, "middleware.json")
	h := buildArrowApp(s, middlewareStack(5)...)
	req := benchRequest(probeRequest(s))
	b.ResetTimer()
	runBenchServeHTTP(b, h, req)
}

func BenchmarkStdlib_Middleware(b *testing.B) {
	s := loadBenchScenario(b, "middleware.json")
	h := buildStdlibMux(b, s)
	req := benchRequest(probeRequest(s))
	b.ResetTimer()
	runBenchServeHTTP(b, h, req)
}

// --- Large: 120-route table ---

func BenchmarkArrow_Large(b *testing.B) {
	s := loadBenchScenario(b, "large.json")
	if len(s.Routes) < 100 {
		b.Fatalf("large corpus: got %d routes, want >= 100", len(s.Routes))
	}
	h := buildArrowApp(s)
	req := benchRequest(probeRequest(s))
	b.ResetTimer()
	runBenchServeHTTP(b, h, req)
}

func BenchmarkStdlib_Large(b *testing.B) {
	s := loadBenchScenario(b, "large.json")
	if len(s.Routes) < 100 {
		b.Fatalf("large corpus: got %d routes, want >= 100", len(s.Routes))
	}
	h := buildStdlibMux(b, s)
	req := benchRequest(probeRequest(s))
	b.ResetTimer()
	runBenchServeHTTP(b, h, req)
}