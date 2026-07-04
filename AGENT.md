# Arrow Agent 使用手册

> **读者**：AI coding agent。本文档用于在 `github.com/fun7257/arrow` 仓库内正确生成、修改、审查 Arrow HTTP 服务代码。  
> **模块**：`github.com/fun7257/arrow` · **Go**：1.22+（仓库当前 `go 1.26.3`）· **零第三方依赖**  
> **人类用户文档**：[`README.md`](README.md) · **性能测试**：[`testdata/bench/README.md`](testdata/bench/README.md)

---

## 1. 30 秒速览

Arrow 是标准库 `net/http` 的增强层，**不替换** `ServeMux`：

- 路由 → 底层 `http.ServeMux`（Go 1.22+ 语法：`GET /posts/{id}`）
- 中间件 → **线性穿透模型**（非洋葱 LIFO）
- Handler 签名 → `func(c *arrow.Context)`，非 `http.HandlerFunc(w,r)`
- 响应辅助 → 子包 `target`（泛型 JSON/XML/错误/分页等）

```go
app := arrow.New()
app.Use(middleware.Recover(), middleware.RequestID(), middleware.Logger())
app.GET("/health", func(c *arrow.Context) { _ = target.OK(c, map[string]string{"status": "ok"}) })
app.ListenAndServe(":8080")
```

---

## 2. 核心语义（必须理解）

### 2.1 穿透管道执行顺序

```
请求 → Pre: M1 → M2 → M3 → Handler → After: M1 → M2 → M3 → 响应
```

| 阶段 | 何时执行 | 如何编写 |
|------|----------|----------|
| **Pre** | 中间件函数体从开始到 `return` | 直接写逻辑；`return` = 穿透下一层 |
| **Handler** | 所有 Pre 完成后 | 路由注册的处理函数 |
| **After** | Handler 完成后，**FIFO**（正向，非洋葱逆序） | `c.After(func(c *arrow.Context) { ... })` |

### 2.2 Abort 规则

| 调用 | 效果 |
|------|------|
| `c.Abort(code)` | 跳过后续 **Pre** 与 **Handler**；已注册的 **After 仍执行** |
| `target.Abort*(c, msg)` | 写 JSON 错误体后 Abort |
| Handler 内写响应但不 Abort | 穿透已完成，After 正常执行 |

**鉴权失败模板**：

```go
func auth(c *arrow.Context) {
    if !valid(c) {
        _ = target.AbortUnauthorized(c, "unauthorized") // 或 c.Abort(401)
        return
    }
}
```

### 2.3 Panic 恢复

注册 `middleware.Recover()` 后，pipeline 在 `defer` 中捕获 panic → 日志 + `500`（若尚未 Abort）。

### 2.4 与洋葱中间件的区别

| | 洋葱 (stdlib) | Arrow 穿透 |
|--|---------------|------------|
| After/Post 顺序 | LIFO 逆序 | **FIFO 正向** |
| 结构 | 嵌套 `next.ServeHTTP` | 扁平 `[]HandlerFunc` |
| 适配 stdlib mw | — | `arrow.Adapt(mw)` |

---

## 3. 仓库导航

```
arrow/
├── arrow.go              # New(), ListenAndServe*, Handler()
├── context.go            # Context, Abort, After, Write, Set/Get
├── router.go             # GET/POST/..., HandleHTTP, Mux()
├── group.go              # Group(prefix) 路由组
├── middleware.go         # Router.Use()
├── pipeline.go           # 穿透执行引擎（内部）
├── adapter.go            # Adapt(), Linear()
├── writer_wrap.go        # ResponseWriter 可选接口委托
├── middleware/           # Recover, RequestID, Logger
├── target/               # 响应写入辅助（见 §6）
├── examples/server/      # 标准示例服务（压测目标）
├── bench_*.go            # 微基准（Arrow vs ServeMux）
├── testdata/bench/       # 基准夹具 JSON
└── scripts/
    ├── run_perf.sh       # 微基准 + 中等压力
    └── stress_test.sh    # 中等压力（examples/server）
```

---

## 4. API 速查

### 4.1 应用与服务器

| API | 说明 |
|-----|------|
| `arrow.New()` | 创建 `*Router`（兼作应用入口） |
| `app.Handler()` | 得到 `http.Handler`，可挂任意 `http.Server` |
| `app.ListenAndServe(addr)` | 快捷 HTTP |
| `app.ListenAndServeTLS(addr, cert, key)` | HTTPS |
| `app.Serve(srv)` / `app.ServeTLS(srv, cert, key)` | 自定义 `http.Server` |

### 4.2 路由注册

| API | 说明 |
|-----|------|
| `GET/POST/PUT/DELETE/PATCH/HEAD/OPTIONS(pattern, h)` | 方法路由 |
| `Any(pattern, h)` | 所有方法 |
| `Handle(method, pattern, h)` | 任意方法；`method==""` 匹配全部 |
| `HandleHTTP(pattern, http.Handler)` | 挂载 stdlib Handler（所有方法） |
| `HandleHTTPMethod(method, pattern, h)` | 挂载 stdlib Handler（指定方法） |
| `Mux()` | 底层 `*http.ServeMux`（高级用法） |
| `Group(prefix)` | 子路由组，继承父级中间件快照 |

**路径参数**：`c.Request.PathValue("id")`（Go 1.22+）

**通配符**：`{id}` 单段 · `{path...}` 剩余路径 · `{$}` 仅尾斜杠 · `/files/` 子树

### 4.3 Context

| 字段/方法 | 说明 |
|-----------|------|
| `c.Writer` | 包装后的 `http.ResponseWriter`（保留 Flusher/Hijacker/Pusher/ReaderFrom） |
| `c.Request` | `*http.Request` |
| `c.After(fn)` | 注册 After 回调（FIFO） |
| `c.Abort(code)` | 终止穿透 |
| `c.IsAborted()` / `c.Written()` / `c.Status()` | 状态查询 |
| `c.Set(key, val)` / `c.Get(key)` | 请求级存储 |
| `c.Write` / `c.WriteHeader` | 直接写响应 |

### 4.4 中间件

```go
app.Use(mw1, mw2)          // 链式，作用于本 Router 及之后创建的子组
api := app.Group("/api")
api.Use(auth)              // 仅 /api 子树
```

| 内置（`middleware` 包） | 阶段 | 说明 |
|-------------------------|------|------|
| `Recover()` | — | 启用 panic → 500 |
| `RequestID()` | Pre | 设置 `X-Request-ID`；`c.Get(middleware.RequestIDKey)` |
| `Logger()` | After | 记录 method/path/status/耗时 |

**自定义中间件模板**：

```go
func Timing() arrow.HandlerFunc {
    return func(c *arrow.Context) {
        start := time.Now()                    // Pre
        c.After(func(c *arrow.Context) {     // After
            log.Printf("%v", time.Since(start))
        })
    }
}
```

### 4.5 标准库互操作

```go
// 经典 func(http.Handler) http.Handler → Arrow
app.Use(arrow.Adapt(stdMiddleware))

// 显式 Pre/Post
app.Use(arrow.Linear(preFn, postFn))

// 静态文件（自动 StripPrefix + 通配符）
app.HandleHTTP("/static/", http.FileServer(http.Dir("./public")))
```

挂载的 `http.Handler` **同样经过**当前 Router 的中间件管道。

---

## 5. 响应编写决策树

```
需要写响应？
├─ 成功 JSON → target.OK / Created / WriteJSON
├─ 错误 JSON（Handler 内，不中断穿透）→ target.BadRequest / NotFound / WriteError
├─ 错误 JSON（中间件/鉴权，中断穿透）→ target.AbortUnauthorized / AbortForbidden / Abort*
├─ 纯文本/HTML → target.WritePlain / WriteHTML
├─ 重定向 → target.Found / SeeOther / WriteRedirect
├─ 文件 → target.WriteFile / WriteAttachment
├─ 流式/SSE → target.WriteStream / WriteSSE
└─ 内容协商 → target.WriteNegotiated
```

| 函数族 | 是否 Abort | 典型场景 |
|--------|------------|----------|
| `target.Write*` / `OK` / `NotFound` | 否 | Handler 正常响应 |
| `target.Abort*` / `AbortWith` | 是 | 中间件鉴权、提前拒绝 |

**注意**：`target.Write*` 在 `c.Written()` 时为 no-op；`Abort*` 在已写入时仅调用 `c.Abort(status)`。

---

## 6. `target` 子包

导入：`import "github.com/fun7257/arrow/target"`

### 6.1 常用写入

```go
target.OK(c, body)                          // 200 JSON
target.Created(c, body)                     // 201 JSON
target.NoContent(c)                         // 204
target.NotFound(c, "not found")             // 404 {"error":"..."}
target.WritePlain(c, 200, "hello\n")        // text/plain
target.WriteJSON(c, status, body)           // 任意状态 JSON
```

### 6.2 中断穿透

```go
target.AbortUnauthorized(c, "unauthorized") // 401 + Abort
target.AbortError(c, 403, "denied")        // 自定义状态
target.AbortWith(c, target.JSON(status, body))
```

### 6.3 数据模型

| 类型 | JSON 形状 |
|------|-----------|
| `target.Error` | `{"error":"message"}` |
| `target.Problem` | RFC 7807 problem+json |
| `target.Page[T]` | `{"items":[],"total":0,"page":1,"size":20}` |
| `target.Envelope[T]` | `{"code":0,"message":"ok","data":{}}` |

### 6.4 全局选项

```go
target.Default.JSONEscapeHTML = true
target.Default.BeforeWrite = func(c *arrow.Context, t target.Target) (target.Target, error) { ... }
target.Default.OnEncodeError = func(c *arrow.Context, err error) { ... }
```

---

## 7. Agent 操作规范

### 7.1 应该做（DO）

1. **新 HTTP 服务**：从 `examples/server/main.go` 复制骨架（Recover + RequestID + Logger）。
2. **路由**：优先 `app.GET/POST`；路径参数用 `{name}` + `PathValue`。
3. **组级中间件**：`api := app.Group("/api"); api.Use(auth)`，不要把鉴权写进每个 handler。
4. **日志/指标**：放在 `c.After`，确保 Abort 后仍能记录。
5. **JSON API**：用 `target` 包，避免手写 `json.Marshal` + `WriteHeader`。
6. **测试**：`go test ./...`；路由行为对照 `stdlib_parity_test.go` 约定。
7. **修改热路径**：保持 `executeZeroMiddleware` 零中间件语义；参考 `router_dispatch_test.go`。

### 7.2 不应该做（DON'T）

1. **不要**假设 After 逆序执行（那是洋葱模型）。
2. **不要**在 Abort 后期望 Handler 仍运行。
3. **不要**在 Pre 阶段依赖 Handler 已设置的 `c.Set` 值。
4. **不要**绕过 Arrow 直接用 `app.Mux().HandleFunc` 注册大量路由（除非有充分理由）；会跳过 Arrow 中间件包装。
5. **不要**引入第三方路由库；Arrow 委托 `ServeMux`。
6. **不要**在 `target.Write*` 之后再次写响应（`Written()` 守卫会忽略）。
7. **不要**删除 `middleware.Recover()` 除非有替代 panic 处理。

### 7.3 常见任务配方

**CRUD API 路由组**：

```go
api := app.Group("/api/v1")
api.Use(requireToken("secret"))
api.GET("/items", listItems)
api.GET("/items/{id}", getItem)
api.POST("/items", createItem)
```

**读取 JSON 请求体**：

```go
func create(c *arrow.Context) {
    var in CreateInput
    if err := json.NewDecoder(c.Request.Body).Decode(&in); err != nil {
        _ = target.BadRequest(c, "invalid json")
        return
    }
    // ...
    _ = target.Created(c, result)
}
```

**Handler 返回 404 但不 Abort**（穿透已完成，适合 Handler 阶段）：

```go
_ = target.NotFound(c, "post not found")
```

**中间件鉴权失败**：

```go
_ = target.AbortUnauthorized(c, "unauthorized")
return
```

---

## 8. 示例服务

路径：`examples/server/main.go`

| 端点 | 方法 | 鉴权 |
|------|------|------|
| `/` | GET | 无 |
| `/health` | GET | 无 |
| `/api/v1/posts` | GET, POST | `Authorization: Bearer demo-token` |
| `/api/v1/posts/{id}` | GET | 同上 |

```bash
go run ./examples/server
curl http://localhost:8080/health
curl -H "Authorization: Bearer demo-token" http://localhost:8080/api/v1/posts
```

---

## 9. 测试与验证

| 命令 | 用途 |
|------|------|
| `go test ./...` | 全量单元/集成测试 |
| `go test -bench=. -benchmem -count=1 -run='^$' ./...` | 微基准（5 场景 Arrow vs stdlib） |
| `./scripts/run_perf.sh` | 微基准 + 中等压力 |
| `./scripts/stress_test.sh` | 仅中等压力（默认 30s×3 端点） |

**关键测试文件**：

| 文件 | 验证内容 |
|------|----------|
| `stdlib_parity_test.go` | 与裸 ServeMux 的 HTTP 可观测行为一致 |
| `router_dispatch_test.go` | 零中间件热路径 + 分发计数器 |
| `pipeline_test.go` | Pre/After/Abort 顺序 |
| `target/target_test.go` | 响应辅助 |

---

## 10. 架构约束（修改代码时）

1. **路由**：`router.go` 将 `GET(pat, h)` 桥接为 `mux.HandleFunc("GET "+pat, ...)`。
2. **零中间件热路径**：无 `app.Use` 时，router 内联 `executeZeroMiddleware`（不经 `pipeline.Run`）。
3. **有中间件**：`pipeline.Run` → Pre 循环 → `finishRequest`（Handler + After FIFO）。
4. **Context 池化**：每请求 `acquireContext` / `releaseContext`（`pool.go`）。
5. **Writer 包装**：按底层接口组合具体类型（`writer_wrap.go`），避免 nil 嵌入接口误满足断言。

---

## 11. 错误排查

| 现象 | 可能原因 |
|------|----------|
| 404/405 与预期不符 | 检查 ServeMux 模式语法；静态路由优先于 `{param}` |
| 中间件未执行 | 路由注册在 `Use` 之前，或注册到了无中间件的兄弟 Group |
| After 顺序不对 | 误以为洋葱 LIFO；Arrow After 为 FIFO |
| Abort 后仍写响应 | `Written()` 已为 true；检查是否重复 Write |
| 鉴权无效 | 用了 `NotFound` 而非 `AbortUnauthorized`（前者不中断穿透） |
| panic 未恢复 | 未注册 `middleware.Recover()` |
| Flusher/Hijacker 不可用 | 底层 `ResponseWriter` 不支持；Arrow 不凭空实现 |

---

## 12. 文档索引

| 文档 | 内容 |
|------|------|
| [`README.md`](README.md) | 人类可读完整文档 |
| [`AGENT.md`](AGENT.md) | 本文档（agent 专用） |
| [`testdata/bench/README.md`](testdata/bench/README.md) | 性能测试说明 |
| [`examples/server/main.go`](examples/server/main.go) | 可运行示例 |

---

*生成维护说明：修改公共 API、中间件语义、target 函数或路由行为时，请同步更新本文件对应章节。*