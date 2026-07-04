# Arrow Penetration Semantics

## Pipeline

```
请求 → Pre: M1 → M2 → M3 → Handler → After: M1 → M2 → M3 → 响应
```

| 阶段 | 编写方式 |
|------|----------|
| Pre | 中间件函数体，`return` 穿透下一层 |
| Handler | 路由注册函数 |
| After | `c.After(func(c *arrow.Context) { ... })`，**FIFO** |

## Abort

| 调用 | 效果 |
|------|------|
| `c.Abort(code)` | 跳过后续 Pre 与 Handler；已注册 After **仍执行** |
| `target.Abort*(c, msg)` | 写 JSON 后 Abort |

```go
func auth(c *arrow.Context) {
    if !valid(c) {
        _ = target.AbortUnauthorized(c, "unauthorized")
        return
    }
}
```

## Panic

`recoverAndRelease` in pipeline/router `defer` → log + 500. Register `middleware.Recover()` by convention.

## vs 洋葱模型

| | 洋葱 | Arrow |
|--|------|-------|
| After | LIFO | **FIFO** |
| 结构 | 嵌套 `next.ServeHTTP` | 扁平 `[]HandlerFunc` |
| stdlib 适配 | — | `arrow.Adapt(mw)` |

## 中间件作用域

```go
app.Use(mw1)                           // 本 Router 路由 + 之后创建的子组
api := app.Group("/api")               // pipe.clone() 快照继承父级中间件
api.Use(auth)
v2 := app.Group("/v2").Use(rateLimit)  // 链式 Use
```

- 父级在 `Group()` **之后**新增的 `Use` **不影响**已创建子组
- 兄弟 Group 中间件互不影响

## 标准库互操作

```go
app.Use(arrow.Adapt(stdMiddleware))
app.Use(arrow.Linear(preFn, postFn))
app.HandleHTTP("/static/", http.FileServer(http.Dir("./public")))
```

`HandleHTTP` / `HandleHTTPMethod` 走 Arrow 中间件管道；`Mux().Handle*` **不走**。

## 热路径（修改框架代码时）

- 无 `app.Use`：router 内联 `executeZeroMiddleware`（不经 `pipeline.Run`）
- 有 `app.Use`：`pipeline.Run` → Pre → `finishRequest`（Handler + After）
- Context 池化：`pool.go`；Writer 包装：`writer_wrap.go`