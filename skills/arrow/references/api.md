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
| `c.Penetrate()` | 语义糖，等价于 `return` |

## Middleware

| 规则 | 说明 |
|------|------|
| 类型 | 仅 `arrow.HandlerFunc` |
| Pre | 函数体直到 `return` |
| After | `c.After(fn)`，FIFO |
| 不支持 | `func(http.Handler) http.Handler` 洋葱包装 |

内置 `middleware` 包：`Recover()` · `RequestID()` · `Logger()`（无 `Auth`）。

## Response decision tree

```
需要写响应？
├─ 成功 JSON → target.OK / Created / WriteJSON
├─ Handler 错误（不 Abort）→ target.WriteJSON(c, status, yourBody)
├─ 中间件鉴权失败 → target.AbortJSON / Abort + target.JSON
├─ RFC 7807 → target.WriteProblem / AbortProblem
├─ 纯文本/HTML → target.WritePlain / WriteHTML
├─ 重定向 → target.Found / SeeOther
├─ 文件 → target.WriteFile / WriteAttachment / WriteFileFS
└─ 流式/SSE → target.WriteStream / WriteSSE
```

| 函数族 | Abort? | 场景 |
|--------|--------|------|
| `target.Write*` / `OK` / `WriteJSON` | 否 | Handler |
| `target.Abort` / `AbortJSON` / `AbortProblem` | 是 | 中间件 |

## target 全部写入函数

### 核心

| 函数 | Abort? |
|------|--------|
| `Write(c, t Target)` | 否 |
| `Abort(c, t Target)` | 是 |
| `Func(fn) Target` | — |
| `JSON(status, body) Target` | — |
| `XML(status, body) Target` | — |

### JSON

| 函数 | Abort? |
|------|--------|
| `WriteJSON(c, status, body)` | 否 |
| `WriteJSONIndent(c, status, body, prefix, indent)` | 否 |
| `WriteEncoded(c, Encoded[T])` | 否 |
| `OK(c, body)` | 否 |
| `Created(c, body)` | 否 |
| `Accepted(c, body)` | 否 |
| `NoContent(c)` | 否 |
| `AbortJSON(c, status, body)` | 是 |

### XML / 文本 / 字节

| 函数 | Abort? |
|------|--------|
| `WriteXML(c, status, body)` | 否 |
| `WritePlain(c, status, body)` | 否 |
| `WriteHTML(c, status, body)` | 否 |
| `WriteBytes(c, status, contentType, body)` | 否 |

### RFC 7807

| 函数 | Abort? |
|------|--------|
| `WriteProblem(c, p Problem)` | 否 |
| `AbortProblem(c, p Problem)` | 是 |

### 重定向

| 函数 | Abort? |
|------|--------|
| `WriteRedirect(c, code, url)` | 否 |
| `MovedPermanently` / `Found` / `SeeOther` | 否 |
| `TemporaryRedirect` / `PermanentRedirect` | 否 |

### 文件 / 流 / 模板

| 函数 | Abort? |
|------|--------|
| `WriteFile` / `WriteAttachment` | 否 |
| `WriteFileFS` / `WriteAttachmentFS` | 否 |
| `WriteStream` / `WriteStreamReader` | 否 |
| `WriteSSE(c, fn)` | 否 |
| `WriteTemplate(c, status, tmpl, data)` | 否 |

### 头部 / 状态

| 函数 | Abort? |
|------|--------|
| `SetHeader` / `SetHeaders` / `SetCookie` | — |
| `WriteWithHeaders(c, t, headers)` | 否 |
| `WriteStatus(c, status)` | 否 |

## target 类型

| 类型 | 说明 |
|------|------|
| `Target` | `Respond(c) error` |
| `Encoded[T]` | status + `Encoder[T]` + body + headers/cookies |
| `Encoder[T]` | `ContentType()` + `Encode(w, v)` |
| `Problem` | RFC 7807（框架唯一提供的业务 JSON 模型） |

```go
target.Default.JSONEscapeHTML = true
target.Default.BeforeWrite = func(c *arrow.Context, t target.Target) (target.Target, error) { ... }
target.Default.OnEncodeError = func(c *arrow.Context, err error) { ... }
```