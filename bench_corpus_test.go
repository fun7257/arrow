// Benchmark corpus loader. Standard fixtures live in testdata/bench/ and follow
// shapes common in Go HTTP framework benchmarks (static REST, param routes,
// GitHub API–style large tables).
package arrow_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
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