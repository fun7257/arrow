# Arrow

基于 Go 标准库 `net/http` 的增强型 HTTP 框架。每一次请求都是一次**穿透**（Penetration），中间件按线性模型执行。

```bash
go get github.com/fun7257/arrow
```

**要求：** Go 1.22+（依赖 `http.ServeMux` 路由增强）

**AI Agent 使用手册：** [`AGENT.md`](AGENT.md)（穿透语义、API 速查、决策树、DO/DON'T）

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
    "github.com/fun7257/arrow"
    "github.com/fun7257/arrow/middleware"
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
        c.Abort(401)
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
api.Use(middleware.Auth())        // 组级中间件
admin := api.Group("/admin")      // 嵌套前缀 /api/admin

api.GET("/posts", list)           // → GET /api/posts
admin.GET("/", dashboard)         // → GET /api/admin/
```

- `Group(prefix)` 返回子 `*Router`，继承父级已注册的中间件
- 子组可继续 `Use()` 追加中间件
- 兄弟组中间件互不影响

---

## 中间件

### 注册

```go
app.Use(middleware.Recover(), middleware.Logger())  // 支持链式调用
```

中间件作用于当前 Router 及其之后创建的子组上注册的所有路由。

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
| Panic | Pipeline 自动恢复，返回 500 |

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

// 直接操作底层 mux
app.Mux().HandleFunc("GET /legacy", legacyHandler)
```

挂载的标准库 Handler 同样经过当前 Router 的中间件管道。

---

## 标准库中间件适配

### Adapt — 经典 `func(http.Handler) http.Handler`

```go
app.Use(arrow.Adapt(stdMiddleware))
```

将标准库洋葱中间件转为穿透模型：Pre 在穿透阶段执行，Post 延迟到 After 阶段。

### Linear — 显式 Pre/Post

```go
app.Use(arrow.Linear(
    func(c *arrow.Context) { /* Pre */ },
    func(c *arrow.Context) { /* After */ },
))
```

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

子包 `github.com/fun7257/arrow/target` 提供泛型优先的 HTTP 响应辅助，采用显式函数风格（无链式 Builder）。

### 核心 API

```go
import "github.com/fun7257/arrow/target"

// 写入 JSON 并继续穿透
target.OK(c, posts)
target.Created(c, post)
target.NotFound(c, "post not found")

// 写入后终止穿透（After 回调仍会执行）
target.AbortUnauthorized(c, "unauthorized")
target.AbortWith(c, target.JSON(http.StatusForbidden, target.Error{Message: "denied"}))
```

`Target` 接口统一响应写入；`Write` 仅写响应，`Abort` / `AbortWith` 写响应后调用 `c.Abort(status)`。

### 泛型编码

```go
target.WriteJSON(c, http.StatusOK, body)
target.WriteJSONAs(c, http.StatusOK, user, func(u User) target.Envelope[User] {
    return target.Envelope[User]{Code: 0, Message: "ok", Data: u}
})
target.OKAs(c, user, func(u User) target.Envelope[User] {
    return target.Envelope[User]{Code: 0, Message: "ok", Data: u}
})
target.WriteEncoded(c, target.Encoded[Post]{
    Status: http.StatusOK, Encoder: target.JSONEncoder[Post]{}, Body: post,
})
target.WriteXML(c, http.StatusOK, payload)
target.WriteNegotiated(c, http.StatusOK, payload) // 按 Accept 选择 JSON/XML
```

内置 `Encoder[T]`：`JSONEncoder`、`XMLEncoder`、`PlainEncoder`；`Encoded[T]` 支持自定义 headers 与 cookies。

### 常用模型

| 类型 | 说明 |
|------|------|
| `target.Error` | `{"error":"message"}` |
| `target.Problem` | RFC 7807 Problem Details |
| `target.Page[T]` | 分页列表 `{items,total,page,size}` |
| `target.Envelope[T]` | 统一包装 `{code,message,data}` |

### 其他能力

- 文本 / HTML / 字节 / 模板：`WritePlain`、`WriteHTML`、`WriteBytes`、`WriteTemplate`
- 重定向：`WriteRedirect`、`Found`、`SeeOther` 等
- 文件：`WriteFile`、`WriteAttachment`、`WriteFileFS`
- 流式 / SSE：`WriteStream(c, status, contentType, fn)`、`WriteStreamReader(c, status, contentType, r)`、`WriteSSE(c, fn func(*EventWriter) error)`
- 头部：`SetHeader`、`SetHeaders`、`SetCookie`、`WriteWithHeaders`

已写入的响应不会重复写入（`c.Written()` 守卫）。

---

## 内置中间件

子包 `github.com/fun7257/arrow/middleware`：

| 中间件 | 阶段 | 说明 |
|--------|------|------|
| `Recover()` | — | 启用 Pipeline panic 恢复（返回 500） |
| `RequestID()` | Pre | 生成或透传 `X-Request-ID` 请求头 |
| `Logger()` | After | 记录 method、path、status、耗时 |

```go
import "github.com/fun7257/arrow/middleware"

app.Use(
    middleware.Recover(),
    middleware.RequestID(),
    middleware.Logger(),
)
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
├── pipeline.go        # 线性穿透执行引擎
├── router.go          # GET/POST/... 路由注册
├── hotpath_dispatch.go # 零中间件热路径计数器（测试钩子）
├── group.go           # 路由组
├── middleware.go      # Use 中间件注册
├── adapter.go         # Adapt / Linear 适配器
├── writer_wrap.go     # ResponseWriter 可选接口委托
├── bench_*.go         # 微基准与夹具测试
├── testdata/bench/    # 基准夹具 JSON（见目录内 README）
├── scripts/
│   ├── run_perf.sh    # 微基准 + 中等压力（推荐入口）
│   └── stress_test.sh # 中等压力（examples/server）
├── examples/server/   # 标准示例服务（压力测试目标）
├── middleware/        # 内置中间件（Recover、Logger、RequestID）
└── target/            # HTTP 响应辅助（泛型 JSON/XML/错误/分页等）
```

---

## 性能测试

两套互补测试，完整说明见 [`testdata/bench/README.md`](testdata/bench/README.md)。

### 一键运行（推荐）

```bash
./scripts/run_perf.sh
```

依次执行微基准与中等压力测试。保存输出：

```bash
BENCH_COUNT=3 OUT_DIR=./perf-out ./scripts/run_perf.sh
# 生成 perf-out/bench.log、perf-out/stress.log
```

### 微基准

Arrow 与 `net/http.ServeMux` 成对对比，计时路径经 `Router` → `Handler()` → `ServeHTTP`：

| 场景 | 说明 |
|------|------|
| minimal | 单路由最小响应 |
| static | 多路由静态表 |
| parametric | `{param}` 路径参数 |
| middleware | 静态表 + 5 层 noop 中间件 |
| large | 120 路由大型表 |

```bash
go test -bench=. -benchmem -count=1 -run='^$' ./...
```

无全局中间件时，路由注册直接调用 `executeZeroMiddleware`（非 `pipeline.Run`）。由 `TestBenchHotPathUsesRouterZeroMiddlewareDispatch` 等测试保障。

### 中等压力

对 `examples/server` 持续压测（默认每端点 **30s**，`/health` 并发 **100**，API 并发 **50**）：

```bash
./scripts/stress_test.sh
```

| 环境变量 | 默认值 | 含义 |
|----------|--------|------|
| `DURATION` | `30s` | 每端点持续时间（hey） |
| `HEALTH_C` | `100` | `/health` 并发 |
| `API_C` | `50` | API 端点并发 |
| `PORT` | `8080` | 服务端口 |
| `START_SERVER` | `1` | 自动启动 `examples/server` |
| `OUT` | — | 将结果写入文件 |

优先使用 [hey](https://github.com/rakyll/hey)；未安装时回退到系统 `ab`。

---

## License

见 [LICENSE](LICENSE) 文件。