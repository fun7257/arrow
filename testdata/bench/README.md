# Arrow HTTP Benchmark Corpus

Standard route and request fixtures for reproducible `go test -bench` runs.
Corpus shapes align with common Go HTTP framework benchmarks (static paths,
REST param routes, GitHub API–style large tables).

## Files

| File | Scenario | Routes | Representative request |
|------|----------|--------|------------------------|
| `minimal.json` | Single-route minimal response | 1 | `GET /ping` |
| `static.json` | Multi-route static table | 12 | `GET /api/v1/users` |
| `parametric.json` | Single- and multi-segment `{param}` | 8 | `GET /repos/octocat/Hello-World` |
| `middleware.json` | Same as static (middleware applied in code) | 12 | `GET /api/v1/users` |
| `large.json` | GitHub REST–style API table | 120 | `GET /repos/golang/go/issues/42` |
| `requests.json` | Index of primary probe request per scenario | — | — |

## Usage

```bash
go test -bench=. -benchmem -count=1 ./...
```

Each benchmark pair (`BenchmarkArrow_*` / `BenchmarkStdlib_*`) shares the same
corpus and issues `ServeHTTP` on the real `app.Handler()` or `http.ServeMux`.