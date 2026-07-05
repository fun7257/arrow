# Arrow Recipes & Troubleshooting

## CRUD API group

```go
api := app.Group("/api/v1")
api.Use(requireToken("secret"))
api.GET("/items", listItems)
api.GET("/items/{id}", getItem)
api.POST("/items", createItem)
```

## Auth middleware

```go
type apiErr struct{ Message string `json:"message"` }

func requireToken(token string) arrow.HandlerFunc {
    return func(c *arrow.Context) {
        if c.Request.Header.Get("Authorization") != "Bearer "+token {
            _ = target.AbortJSON(c, http.StatusUnauthorized, apiErr{Message: "unauthorized"})
            return
        }
    }
}
```

## JSON body

```go
func create(c *arrow.Context) {
    var in CreateInput
    if err := json.NewDecoder(c.Request.Body).Decode(&in); err != nil {
        _ = target.WriteJSON(c, http.StatusBadRequest, apiErr{Message: "invalid json"})
        return
    }
    _ = target.Created(c, result)
}
```

## Custom middleware (timing)

```go
func Timing() arrow.HandlerFunc {
    return func(c *arrow.Context) {
        start := time.Now()
        c.After(func(c *arrow.Context) {
            log.Printf("%v", time.Since(start))
        })
    }
}
```

## examples/server

| 端点 | 鉴权 |
|------|------|
| `GET /health` | 无 |
| `GET/POST /api/v1/posts` | `Bearer demo-token` |
| `GET /api/v1/posts/{id}` | 同上 |

```bash
go run ./examples/server
curl http://localhost:8080/health
```

## Testing

| 命令 | 用途 |
|------|------|
| `go test ./...` | 全量测试 |
| `go test -bench=. -benchmem -count=1 -run='^$' ./...` | 微基准 |
| `./scripts/run_perf.sh` | 微基准 + 中等压力 |

| 测试文件 | 验证 |
|----------|------|
| `stdlib_parity_test.go` | Arrow vs ServeMux 路由基线 |
| `stdlib_inherit_test.go` | 路由/Writer 行为与标准库一致 |
| `router_dispatch_test.go` | 零中间件热路径 |
| `pipeline_test.go` | Pre/After/Abort FIFO |
| `target/target_test.go` | `target` 写入与 Abort |

## Troubleshooting

| 现象 | 原因 |
|------|------|
| 404/405 异常 | ServeMux 模式；静态路由优先 `{param}` |
| 中间件未跑 | 路由在 `Use` 前注册，或兄弟 Group |
| After 顺序错 | 误用洋葱 LIFO |
| 鉴权无效 | Handler 里 `WriteJSON` 而非中间件里 `AbortJSON` |
| 重复写响应 | `c.Written()` 已为 true |
| Flusher 不可用 | 底层 Writer 不支持 |

## 文档索引

| 文件 | 受众 |
|------|------|
| `README.md` | 人类用户 |
| `skills/arrow/SKILL.md` | AI agent（本 skill 源文件） |
| `skills/README.md` | Skill 安装说明 |
| `testdata/bench/README.md` | 性能测试 |