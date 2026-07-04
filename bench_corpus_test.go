// Benchmark corpus loader. Standard fixtures live in testdata/bench/ and follow
// shapes common in Go HTTP framework benchmarks (static REST, param routes,
// GitHub API–style large tables).
package arrow_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fun7257/arrow"
)

// BenchRoute is a single route entry in the benchmark corpus.
type BenchRoute struct {
	Method   string `json:"method"`
	Pattern  string `json:"pattern"`
	Response string `json:"response"`
}

// BenchRequest is a representative HTTP request sample.
type BenchRequest struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Host   string `json:"host,omitempty"`
}

// BenchScenario is a named benchmark scenario with routes and probe requests.
type BenchScenario struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Routes      []BenchRoute   `json:"routes"`
	Requests    []BenchRequest `json:"requests"`
}

func benchDataDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "testdata/bench"
	}
	return filepath.Join(filepath.Dir(file), "testdata", "bench")
}

func loadBenchScenario(tb testing.TB, filename string) BenchScenario {
	tb.Helper()
	path := filepath.Join(benchDataDir(), filename)
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read corpus %s: %v", filename, err)
	}
	var s BenchScenario
	if err := json.Unmarshal(data, &s); err != nil {
		tb.Fatalf("parse corpus %s: %v", filename, err)
	}
	if len(s.Routes) == 0 {
		tb.Fatalf("corpus %s: no routes", filename)
	}
	if len(s.Requests) == 0 {
		tb.Fatalf("corpus %s: no requests", filename)
	}
	return s
}

func probeRequest(s BenchScenario) BenchRequest {
	return s.Requests[0]
}

// BenchProbeIndex maps scenario names to primary probe requests (testdata/bench/requests.json).
type BenchProbeIndex struct {
	Description string                       `json:"description"`
	Probes      map[string]BenchProbeEntry   `json:"probes"`
}

type BenchProbeEntry struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Corpus string `json:"corpus"`
}

func loadBenchProbeIndex(tb testing.TB) BenchProbeIndex {
	tb.Helper()
	path := filepath.Join(benchDataDir(), "requests.json")
	data, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read probe index: %v", err)
	}
	var idx BenchProbeIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		tb.Fatalf("parse probe index: %v", err)
	}
	return idx
}

func TestBenchProbeRequestsAlignWithCorpus(t *testing.T) {
	idx := loadBenchProbeIndex(t)
	for name, probe := range idx.Probes {
		s := loadBenchScenario(t, probe.Corpus)
		if s.Name != "" && s.Name != name {
			t.Errorf("%s: corpus name %q != probe key %q", probe.Corpus, s.Name, name)
		}
		first := probeRequest(s)
		if first.Method != probe.Method || first.Path != probe.Path {
			t.Errorf("%s: corpus probe %+v != index %+v", name, first, probe)
		}
	}
}

func TestBenchHotPathUsesHandler(t *testing.T) {
	s := loadBenchScenario(t, "minimal.json")
	wantBody := s.Routes[0].Response
	req := benchRequest(probeRequest(s))

	h := buildArrowApp(s)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("bench handler status = %d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != wantBody {
		t.Fatalf("bench handler body = %q, want %q", got, wantBody)
	}

	// buildArrowApp must not register global middleware; a Use'd app must differ.
	mwRan := false
	appWithMW := arrow.New()
	appWithMW.Use(func(c *arrow.Context) { mwRan = true })
	registerArrowRoutes(appWithMW, s.Routes)
	recMW := httptest.NewRecorder()
	appWithMW.Handler().ServeHTTP(recMW, req)
	if !mwRan {
		t.Fatal("middleware must run when app.Use is called")
	}
	if recMW.Body.String() != wantBody {
		t.Fatalf("middleware app body = %q, want %q", recMW.Body.String(), wantBody)
	}

	// Zero-middleware bench path uses serveZeroMiddlewareFromHTTP (see router_dispatch_test.go).
	var afterRan bool
	appAfter := arrow.New()
	appAfter.GET(s.Routes[0].Pattern, func(c *arrow.Context) {
		c.After(func(c *arrow.Context) { afterRan = true })
		c.Write([]byte(wantBody))
	})
	recAfter := httptest.NewRecorder()
	appAfter.Handler().ServeHTTP(recAfter, req)
	if !afterRan {
		t.Fatal("zero-middleware inline router path must execute After callbacks")
	}
	if recAfter.Body.String() != wantBody {
		t.Fatalf("after app body = %q, want %q", recAfter.Body.String(), wantBody)
	}
}

func TestBenchCorpusLoads(t *testing.T) {
	cases := []struct {
		file      string
		minRoutes int
	}{
		{"minimal.json", 1},
		{"static.json", 10},
		{"parametric.json", 5},
		{"middleware.json", 10},
		{"large.json", 100},
	}
	for _, tc := range cases {
		s := loadBenchScenario(t, tc.file)
		if len(s.Routes) < tc.minRoutes {
			t.Errorf("%s: got %d routes, want >= %d", tc.file, len(s.Routes), tc.minRoutes)
		}
	}
}