# Arrow

基于 Go 标准库 `net/http` 的增强型 HTTP 框架。每一次请求都是一次**穿透**（Penetration），中间件按线性模型执行。

```bash
go get github.com/fun7257/arrow
```

**要求：** Go 1.22+（依赖 `http.ServeMux` 路由增强）

---

## 核心理念：穿透线性模型

Arrow 不使用经典洋葱模型的嵌套闭包，而是扁平的穿透管道：

```
请求进入
  → Pre：M1 → M2 → M3（按注册顺序）
  → Handler：路由处理函数
  → After：M1 → M2 → M3（按注册顺序，正向 FIFO）
  → 响应返回
```

| 维度 | 洋葱模型 | Arrow 穿透模型 |
|------|----------|----------------|
| Pre 执行 | 注册顺序正向 | 注册顺序正向 |
| After 执行 | **逆序**（LIFO） | **正向**（FIFO） |
| 结构 | 嵌套闭包 | 扁平 Pipeline |

**自动区分 Pre / After：**

- 中间件函数体执行期间 = **Pre 阶段**（`return` 即穿透下一层）
- `c.After(fn)` 注册的回调 = **After 阶段**（Handler 完成后执行）

---

## 快速开始

```go
package main

import (
    "net/http"

    "github.com/fun7257/arrow"
    "github.com/fun7257/arrow/middleware"
    "github.com/fun7257/arrow/target"
)

func main() {
    app := arrow.New()

    app.Use(middleware.Recover())
    app.Use(middleware.RequestID())
    app.Use(middleware.Logger())

    app.GET("/", home)
    app.GET("/health", health)

    api := app.Group("/api")
    api.Use(auth)
    api.GET("/posts", listPosts)
    api.GET("/posts/{id}", showPost)
    api.POST("/posts", createPost)

    app.ListenAndServe(":8080")
}

func showPost(c *arrow.Context) {
    id := c.Request.PathValue("id")
    c.Write([]byte(id))
}

func auth(c *arrow.Context) {
    if c.Request.Header.Get("Authorization") == "" {
        _ = target.AbortJSON(c, http.StatusUnauthorized, map[string]string{"message": "unauthorized"})
        return
    }
}
```

---

## 路由

路由基于 Go 1.22+ `http.ServeMux`，完整支持标准库路由语法。

### HTTP 方法注册

| 方法 | 说明 |
|------|------|
| `GET(pattern, handler)` | GET 请求 |
| `POST(pattern, handler)` | POST 请求 |
| `PUT(pattern, handler)` | PUT 请求 |
| `DELETE(pattern, handler)` | DELETE 请求 |
| `PATCH(pattern, handler)` | PATCH 请求 |
| `HEAD(pattern, handler)` | HEAD 请求 |
| `OPTIONS(pattern, handler)` | OPTIONS 请求 |
| `Any(pattern, handler)` | 匹配所有 HTTP 方法 |
| `Handle(method, pattern, handler)` | 任意方法；`method` 为空时匹配所有方法 |

内部将 `GET("/posts/{id}", h)` 桥接为 ServeMux 模式 `GET /posts/{id}`。

### 通配符与路径参数

| 语法 | 示例 | 说明 |
|------|------|------|
| `{id}` | `/posts/{id}` | 匹配单个路径段，通过 `c.Request.PathValue("id")` 获取 |
| `{path...}` | `/files/{path...}` | 匹配剩余所有路径段 |
| `{$}` | `/posts/{$}` | 仅匹配带尾斜杠的路径 `/posts/` |
| 尾斜杠 | `/files/` | 匹配以 `/files/` 开头的所有子路径 |

### Host 路由

```go
app.Handle("GET", "example.com/api", handler)
```

### 路由优先级

遵循标准库 most-specific-wins 规则。例如 `/posts/latest` 优先于 `/posts/{id}`。

未匹配返回 **404**，方法不匹配返回 **405**（含 `Allow` 头）。

---

## 路由组

```go
api := app.Group("/api")          // 路径前缀 /api
api.Use(auth)                     // 组级中间件（自定义 HandlerFunc）
admin := api.Group("/admin")      // 嵌套前缀 /api/admin

api.GET("/posts", list)           // → GET /api/posts
admin.GET("/", dashboard)         // → GET /api/admin/

v2 := app.Group("/api/v2")
v2.Use(rateLimit)
v2.GET("/status", status)
```

- `Group(prefix)` 返回子路由作用域，**创建时**继承父级已注册的中间件（`pipe.clone()` 快照）
- 组级中间件须先赋值再 `Use`：`api := app.Group("/api"); api.Use(auth)`（Group 与 Use 不能写在同一表达式）
- 子组可继续 `Use()` 追加中间件；**之后**在父级新增的 `Use` 不会影响已创建的子组
- 兄弟组中间件互不影响（见 `router_test.go`）

---

## 中间件

### 注册

```go
app.Use(middleware.Recover())
app.Use(middleware.Logger())
```

经典显式注册：每次 `Use` 只接受一个中间件，无返回值。不可 `app.Use(a).Use(b)` 链式串联；路由组须先 `api := app.Group(prefix)` 再 `api.Use(mw)`（`Group` 与 `Use` 不能写在同一表达式）。

中间件作用于：**在当前 Router 上注册的路由**，以及 **在此之后** 用该 Router 作为父级创建的子组上的路由。

内置中间件包 `middleware` **仅提供** `Recover`、`RequestID`、`Logger` 三个函数；鉴权等业务中间件需自行实现（见上文 `auth` 示例）。

中间件须为 `arrow.HandlerFunc`（函数体 = Pre，`c.After` = After）。**不支持**洋葱式 `func(http.Handler) http.Handler` 包装。

### 自定义中间件

```go
func Timing() arrow.HandlerFunc {
    return func(c *arrow.Context) {
        start := time.Now()                    // Pre 阶段

        c.After(func(c *arrow.Context) {       // After 阶段
            log.Printf("took %v", time.Since(start))
        })
    }
}
```

### Abort 语义

```go
func Auth() arrow.HandlerFunc {
    return func(c *arrow.Context) {
        if !valid(c) {
            c.Abort(401)   // 跳过后续 Pre 和 Handler
            return
        }
    }
}
```

| 行为 | 说明 |
|------|------|
| `Abort(code)` | 终止穿透，跳过后续 Pre 中间件和 Handler |
| 已注册的 `After` | **仍然执行**（日志、指标不丢失） |
| Panic | `defer recoverAndRelease` 自动恢复，返回 500（建议注册 `middleware.Recover()`） |

---

## Context API

| 方法 | 说明 |
|------|------|
| `c.Writer` | `http.ResponseWriter`（支持 Flusher / Hijacker / Pusher / ReaderFrom 委托） |
| `c.Request` | `*http.Request`（含 `PathValue()`） |
| `c.After(fn)` | 注册 After 回调 |
| `c.Abort(code)` | 终止穿透 |
| `c.Penetrate()` | 显式穿透标记（语义糖，等价于 `return`） |
| `c.IsAborted()` | 是否已终止 |
| `c.Status()` | 响应状态码 |
| `c.Set(key, val)` / `c.Get(key)` | 请求级键值存储 |
| `c.Write(b)` / `c.WriteHeader(code)` | 写响应 |

---

## 标准库 Handler 挂载

| 方法 | 说明 |
|------|------|
| `HandleHTTP(pattern, h)` | 挂载 `http.Handler`，匹配所有方法 |
| `HandleHTTPMethod(method, pattern, h)` | 挂载 `http.Handler`，指定方法 |
| `Mux()` | 暴露底层 `*http.ServeMux`，支持高级定制 |

```go
// 静态文件（自动适配 Go 1.22+ 通配符 + StripPrefix）
app.HandleHTTP("/static/", http.FileServer(http.Dir("./public")))

// 高级：直接操作底层 mux（跳过 Arrow 中间件与 Context 包装，慎用）
app.Mux().HandleFunc("GET /legacy", legacyHandler)
```

通过 `HandleHTTP` / `HandleHTTPMethod` 挂载的标准库 Handler **经过**当前 Router 的中间件管道；直接 `Mux().Handle*` 则与裸 `ServeMux` 行为一致。

中间件须实现为 `arrow.HandlerFunc`（函数体 = Pre，`c.After` = After）。**不提供**洋葱式 `func(http.Handler) http.Handler` 的适配层。

---

## 服务器启动

| 方法 | 说明 |
|------|------|
| `ListenAndServe(addr)` | HTTP 服务 |
| `ListenAndServeTLS(addr, cert, key)` | HTTPS 服务 |
| `Serve(srv)` | 使用自定义 `http.Server` |
| `ServeTLS(srv, cert, key)` | 自定义 Server + TLS |
| `Handler()` | 返回 `http.Handler`，可挂到任意 Server |

```go
app := arrow.New()
// ...注册路由

// 方式一：快捷启动
app.ListenAndServe(":8080")

// 方式二：自定义 Server
srv := &http.Server{
    Addr:         ":8080",
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
}
app.Serve(srv)

// 方式三：作为 Handler 使用
http.ListenAndServe(":8080", app.Handler())
```

---

## 响应辅助（target）

子包 `github.com/fun7257/arrow/target` 负责**写入开发者提供的响应**，不规定业务 JSON 形状。采用显式函数风格（无链式 Builder）。

### 核心 API

```go
import "github.com/fun7257/arrow/target"

// 写入 JSON 并继续穿透（body 为开发者自定义类型）
target.OK(c, posts)
target.Created(c, post)
target.WriteJSON(c, http.StatusNotFound, apiErr{Message: "post not found"})

// 写入后终止穿透（After 回调仍会执行，用于中间件鉴权等）
target.AbortJSON(c, http.StatusUnauthorized, apiErr{Message: "unauthorized"})
target.Abort(c, target.JSON(http.StatusForbidden, body))
```

`Target` 接口统一响应写入；`Write` 仅写响应，`Abort` / `AbortJSON` 写响应后调用 `c.Abort(status)`。

### 泛型编码

```go
target.WriteJSON(c, http.StatusOK, body)
target.WriteEncoded(c, target.Encoded[Post]{
    Status: http.StatusOK, Encoder: target.JSONEncoder[Post]{}, Body: post,
})
target.WriteXML(c, http.StatusOK, payload)
```

内置 `Encoder[T]`：`JSONEncoder`、`XMLEncoder`、`PlainEncoder`；`Encoded[T]` 支持自定义 headers 与 cookies。

### 行业标准

| 类型 | 说明 |
|------|------|
| `target.Problem` | RFC 7807 Problem Details（`WriteProblem` / `AbortProblem`） |

### 其他能力

- 文本 / HTML / 字节 / 模板：`WritePlain`、`WriteHTML`、`WriteBytes`、`WriteTemplate`
- 重定向：`WriteRedirect`、`Found`、`SeeOther` 等
- 文件：`WriteFile`、`WriteAttachment`、`WriteFileFS`、`WriteAttachmentFS`
- Problem Details：`WriteProblem`、`AbortProblem`
- 流式 / SSE：`WriteStream(c, status, contentType, fn)`、`WriteStreamReader(c, status, contentType, r)`、`WriteSSE(c, fn func(*EventWriter) error)`
- 头部：`SetHeader`、`SetHeaders`、`SetCookie`、`WriteWithHeaders`

已写入的响应不会重复写入（`c.Written()` 守卫）。

---

## 内置中间件

子包 `github.com/fun7257/arrow/middleware`：

| 中间件 | 阶段 | 说明 |
|--------|------|------|
| `Recover()` | — | 惯例性注册；实际 panic 恢复由 pipeline/router 的 `recoverAndRelease` 执行 |
| `RequestID()` | Pre | 生成或透传 `X-Request-ID` 请求头 |
| `Logger()` | After | 记录 method、path、status、耗时 |

```go
import "github.com/fun7257/arrow/middleware"

app.Use(middleware.Recover())
app.Use(middleware.RequestID())
app.Use(middleware.Logger())
```

---

## ResponseWriter 兼容性

Arrow 通过组合类型包装 `ResponseWriter`，仅在底层支持时暴露可选接口，与标准库类型断言行为一致：

| 接口 | 支持 |
|------|------|
| `http.Flusher` | 底层支持时可用（SSE、流式响应） |
| `http.Hijacker` | 底层支持时可用（WebSocket） |
| `http.Pusher` | 底层支持时可用（HTTP/2 Server Push） |
| `io.ReaderFrom` | 底层支持时可用（sendfile 优化） |
| `http.ResponseController` | 通过 `Unwrap()` 链支持 |

---

## 与标准库的关系

Arrow **不替换** `net/http`，而是增强它：

- 路由匹配完全委托 `http.ServeMux`
- `Engine` 实现 `http.Handler`
- 404 / 405 / 路由优先级等行为与标准库一致
- 零第三方依赖

---

## 项目结构

```
arrow/
├── arrow.go           # 应用入口、服务器启动
├── context.go         # Context、Abort、After
├── pipeline.go        # 线性穿透执行引擎（executeZeroMiddleware）
├── router.go          # GET/POST/... 路由注册
├── group.go           # 路由组（pipe.clone 继承中间件）
├── middleware.go      # Use 中间件注册

├── writer_wrap.go     # ResponseWriter 可选接口委托
├── middleware/        # 内置中间件（Recover、RequestID、Logger）
└── target/            # HTTP 响应写入（开发者自定 body；RFC 7807 Problem）
```

---

## License

见 [LICENSE](LICENSE) 文件。