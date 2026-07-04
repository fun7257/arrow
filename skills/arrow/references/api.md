# Arrow API Reference

## Application

| API | 说明 |
|-----|------|
| `arrow.New()` | 创建 `*Router` |
| `app.Handler()` | `http.Handler` |
| `app.ListenAndServe(addr)` | HTTP |
| `app.ListenAndServeTLS(addr, cert, key)` | HTTPS |
| `app.Serve(srv)` / `ServeTLS` | 自定义 `http.Server` |

## Routing

| API | 说明 |
|-----|------|
| `GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS(pat, h)` | 方法路由 |
| `Any(pat, h)` | 所有方法 |
| `Handle(method, pat, h)` | 任意方法；`method==""` 匹配全部 |
| `HandleHTTP(pat, http.Handler)` | stdlib Handler，所有方法 |
| `HandleHTTPMethod(method, pat, h)` | stdlib Handler，指定方法 |
| `Group(prefix)` | 子路由组 |
| `Mux()` | 裸 `*http.ServeMux`（跳过 Arrow 包装） |

**PathValue**: `c.Request.PathValue("id")`  
**Wildcards**: `{id}` · `{path...}` · `{$}` · `/files/` subtree

## Context

| API | 说明 |
|-----|------|
| `c.Writer` / `c.Request` | 包装 ResponseWriter + Request |
| `c.After(fn)` | After 回调（FIFO） |
| `c.Abort(code)` | 终止穿透 |
| `c.IsAborted()` / `c.Written()` / `c.Status()` | 状态 |
| `c.Set` / `c.Get` | 请求级 KV |
| `c.Write` / `c.WriteHeader` | 写响应 |

## Builtin middleware

| 函数 | 阶段 | 说明 |
|------|------|------|
| `Recover()` | — | 惯例；panic 由 `recoverAndRelease` 处理 |
| `RequestID()` | Pre | `X-Request-ID`；`middleware.RequestIDKey` |
| `Logger()` | After | method/path/status/耗时 |

## Response decision tree

```
需要写响应？
├─ 成功 JSON → target.OK / Created / WriteJSON
├─ Handler 错误（不 Abort）→ target.BadRequest / NotFound / WriteError
├─ 中间件鉴权失败 → target.AbortUnauthorized / AbortForbidden / Abort*
├─ 纯文本/HTML → target.WritePlain / WriteHTML
├─ 重定向 → target.Found / SeeOther
├─ 文件 → target.WriteFile / WriteAttachment / WriteFileFS
├─ 流式/SSE → target.WriteStream / WriteSSE
└─ 协商 → target.WriteNegotiated
```

| 函数族 | Abort? | 场景 |
|--------|--------|------|
| `target.Write*` / `OK` / `NotFound` | 否 | Handler |
| `target.Abort*` / `AbortWith` | 是 | 中间件 |

## target 常用

```go
target.OK(c, body)
target.Created(c, body)
target.NoContent(c)
target.NotFound(c, "msg")
target.AbortUnauthorized(c, "msg")
target.AbortWith(c, target.JSON(status, body))
target.WritePlain(c, 200, "text\n")
target.OKEnvelope(c, data)
target.WriteProblem(c, problem)
```

| 类型 | JSON |
|------|------|
| `Error` | `{"error":"..."}` |
| `Problem` | RFC 7807 |
| `Page[T]` | `{items,total,page,size}` |
| `Envelope[T]` | `{code,message,data}` |

```go
target.Default.JSONEscapeHTML = true
target.Default.BeforeWrite = func(c *arrow.Context, t target.Target) (target.Target, error) { ... }
```