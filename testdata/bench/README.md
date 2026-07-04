# Arrow HTTP Benchmark Corpus

Standard route and request fixtures for reproducible performance testing.

## Layout

| Layer | Location | Purpose |
|-------|----------|---------|
| Micro-benchmarks | `bench_test.go`, `bench_build_test.go` | Arrow vs `net/http.ServeMux`, `ns/op` + allocs |
| Corpus + probes | `testdata/bench/*.json` | Shared route tables and representative requests |
| Corpus tests | `bench_corpus_test.go` | Fixture counts, probe alignment, handler smoke test |
| Hot-path tests | `router_dispatch_test.go`, `router_source_test.go`, `pipeline_internal_test.go` | Zero-mw dispatch counters, source checks, semantics |
| Medium stress | `scripts/stress_test.sh` | Sustained load on `examples/server` (~30s × 3 endpoints) |
| Full suite | `scripts/run_perf.sh` | Micro-benchmarks + medium stress in one command |

## Corpus files

| File | Scenario | Routes | Representative request |
|------|----------|--------|------------------------|
| `minimal.json` | Single-route minimal response | 1 | `GET /ping` |
| `static.json` | Multi-route static table | 12 | `GET /api/v1/users` |
| `parametric.json` | Single- and multi-segment `{param}` | 8 | `GET /users/octocat` |
| `middleware.json` | Same as static (middleware applied in code) | 12 | `GET /api/v1/users` |
| `large.json` | GitHub REST–style API table | 120 | `GET /repos/golang/go/issues/42` |
| `requests.json` | Index of primary probe request per scenario | — | — |

## Run micro-benchmarks

```bash
go test -bench=. -benchmem -count=1 -run='^$' ./...
```

Each pair (`BenchmarkArrow_*` / `BenchmarkStdlib_*`) shares the same corpus and
calls `app.Handler().ServeHTTP` or `http.ServeMux.ServeHTTP`.

Zero-middleware scenarios use `executeZeroMiddleware` via router registration.
Verified by `TestBenchHotPathUsesRouterZeroMiddlewareDispatch` and
`TestRouterZeroMiddlewareUsesExecuteZeroMiddleware`.

## Run medium stress test

```bash
./scripts/stress_test.sh
# or micro + stress together:
./scripts/run_perf.sh
```

Default load: **30s sustained**, concurrency **100** on `/health` and **50** on
authenticated API routes. Uses `hey` when installed, otherwise `ab`.

Probe requests in `requests.json` must match each corpus file's first `requests`
entry (`TestBenchProbeRequestsAlignWithCorpus`).