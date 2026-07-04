---
name: arrow
description: >
  Develop, review, and modify code using the Arrow HTTP framework
  (github.com/fun7257/arrow): linear penetration middleware, Context/Abort/After,
  target response helpers, ServeMux routing. Read this skill when working in the
  arrow repo, adding routes/middleware/handlers, using the target package,
  integrating net/http handlers, or when the user mentions Arrow, 穿透, or
  /arrow. Human docs: README.md. Deep reference: references/ in this skill dir.
metadata:
  short-description: "Arrow HTTP framework agent guide"
---

# Arrow HTTP Framework

Module `github.com/fun7257/arrow` · Go 1.22+ · zero third-party deps · delegates routing to `http.ServeMux`.

## When this skill applies

- Creating or editing HTTP services, routes, middleware, handlers in this repo
- Using `target` for JSON/errors/responses
- Reviewing Arrow-specific semantics (Abort, After FIFO, penetration)
- Performance work: `testdata/bench/README.md`, `./scripts/run_perf.sh`

## Critical semantics (never violate)

**Execution order** (NOT onion/LIFO):

```
Pre: M1 → M2 → M3 → Handler → After: M1 → M2 → M3 (FIFO)
```

| Rule | Detail |
|------|--------|
| Pre | Middleware function body until `return` |
| After | `c.After(fn)` — runs **forward FIFO** after Handler |
| `c.Abort(code)` | Skips remaining Pre + Handler; **After still runs** |
| `target.Abort*(c, …)` | Writes body then Abort — use in **middleware/auth** |
| `target.NotFound` etc. | Handler response **without** Abort |
| After order | **Never** assume LIFO (that is stdlib onion) |

**Builtin middleware** (`middleware` package only): `Recover()`, `RequestID()`, `Logger()`. **No** `middleware.Auth()` — implement auth yourself.

**Routing**: `app.GET("/posts/{id}", h)` → `c.Request.PathValue("id")`. Do not add third-party routers.

**Groups**: `Group(prefix)` clones parent middleware at creation time; parent `Use` after `Group()` does not affect existing child groups.

**Mux bypass**: `app.Mux().Handle*` skips Arrow middleware and Context — avoid unless intentional.

## Default service skeleton

Copy from `examples/server/main.go`:

```go
app := arrow.New()
app.Use(middleware.Recover(), middleware.RequestID(), middleware.Logger())
app.GET("/health", func(c *arrow.Context) {
    _ = target.OK(c, map[string]string{"status": "ok"})
})
app.ListenAndServe(":8080")
```

## Response decision (quick)

| Situation | Use |
|-----------|-----|
| Success JSON | `target.OK` / `Created` / `WriteJSON` |
| Handler error (no stop) | `target.BadRequest` / `NotFound` |
| Middleware auth fail | `target.AbortUnauthorized` + `return` |
| Plain text | `target.WritePlain` |

Full tree: [references/api.md](references/api.md#response-decision-tree)

## Workflow for agents

1. **Read context**: `examples/server/main.go` for patterns; `README.md` for human-facing API.
2. **Implement**: `app.GET/POST` + `target.*`; group middleware via `api := app.Group("/api"); api.Use(auth)`.
3. **Log/metrics**: register in `c.After` so Abort does not skip them.
4. **Test**: `go test ./...` before finishing; route parity expectations in `stdlib_parity_test.go`.
5. **Hot-path changes**: preserve `executeZeroMiddleware` zero-mw semantics; see `router_dispatch_test.go`.

## DO / DON'T

**DO**: `target` for JSON APIs · group-level auth · `HandleHTTP` for stdlib handlers · `arrow.Adapt` for stdlib middleware

**DON'T**: onion After assumptions · `Mux().HandleFunc` for app routes · `middleware.Auth()` · double-write after `target.Write*` · third-party routers

## Reference files (read on demand)

| File | Content |
|------|---------|
| [references/semantics.md](references/semantics.md) | Penetration model, Abort, panic, stdlib interop |
| [references/api.md](references/api.md) | Router/Context/target API tables |
| [references/recipes.md](references/recipes.md) | CRUD, auth, testing, troubleshooting |

## Repo map

```
arrow.go router.go context.go pipeline.go group.go adapter.go
middleware/   → Recover, RequestID, Logger
target/       → response helpers
examples/server/ → canonical app
testdata/bench/  → perf fixtures
scripts/run_perf.sh · stress_test.sh
```

## Maintenance

When changing public API, middleware semantics, or `target` functions, update this skill and `references/` together with `README.md`.