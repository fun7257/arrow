# Arrow HTTP 性能测试说明

可复现的性能测试分为两层：**微基准**（`go test -bench`）与**中等压力**（`examples/server` + `scripts/stress_test.sh`）。

## 目录结构

| 层级 | 位置 | 作用 |
|------|------|------|
| 微基准 | `bench_test.go`、`bench_build_test.go` | Arrow vs `net/http.ServeMux`，输出 `ns/op`、allocs |
| 夹具 | `testdata/bench/*.json` | 各场景路由表与探针请求 |
| 夹具测试 | `bench_corpus_test.go` | 路由数量、探针对齐、handler 冒烟 |
| 热路径验证 | `router_dispatch_test.go`、`router_source_test.go`、`pipeline_internal_test.go` | 零中间件分发计数、源码检查、语义等价 |
| 中等压力 | `scripts/stress_test.sh` | 对 `examples/server` 持续压测（默认 30s × 3 端点） |
| 完整套件 | `scripts/run_perf.sh` | 微基准 + 中等压力 |

## 微基准场景

| 夹具 | 场景 | 路由数 | 探针请求 | 基准函数 |
|------|------|--------|----------|----------|
| `minimal.json` | 单路由最小响应 | 1 | `GET /ping` | `BenchmarkArrow_Minimal` / `BenchmarkStdlib_Minimal` |
| `static.json` | 多路由静态表 | 12 | `GET /api/v1/users` | `BenchmarkArrow_Static` / `BenchmarkStdlib_Static` |
| `parametric.json` | `{param}` 路由 | 8 | `GET /users/octocat` | `BenchmarkArrow_Parametric` / `BenchmarkStdlib_Parametric` |
| `middleware.json` | 静态表 + 代码层中间件 | 12 | `GET /api/v1/users` | `BenchmarkArrow_Middleware` / `BenchmarkStdlib_Middleware` |
| `large.json` | GitHub REST 风格大表 | 120 | `GET /repos/golang/go/issues/42` | `BenchmarkArrow_Large` / `BenchmarkStdlib_Large` |
| `requests.json` | 各场景主探针索引 | — | — | `TestBenchProbeRequestsAlignWithCorpus` |

### 运行

```bash
# 仅微基准
go test -bench=. -benchmem -count=1 -run='^$' ./...

# 微基准 + 中等压力
./scripts/run_perf.sh
```

每对基准共享同一夹具，在 `app.Handler()` 或 `http.ServeMux` 上调用 `ServeHTTP`。

**热路径**：minimal / static / parametric / large 场景无全局中间件，路由注册内联
`executeZeroMiddleware`（不经 `pipeline.Run` / `runNoMiddleware`）。验证测试：

- `TestBenchHotPathUsesRouterZeroMiddlewareDispatch` — 运行时计数器
- `TestRouterZeroMiddlewareUsesExecuteZeroMiddleware` — `router.go` 源码检查
- `TestServeZeroMiddlewareFromHTTPEquivalentToRunNoMiddleware` — 行为等价

## 中等压力测试

目标进程：`examples/server`（Recover + RequestID + Logger 中间件）。

```bash
./scripts/stress_test.sh
```

默认压测三个端点：

| 端点 | 时长 | 并发 | 说明 |
|------|------|------|------|
| `GET /health` | 30s | 100 | 健康检查 JSON |
| `GET /api/v1/posts` | 30s | 50 | 需 `Authorization: Bearer demo-token` |
| `GET /api/v1/posts/1` | 30s | 50 | 参数路由 + 鉴权 |

### 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `DURATION` | `30s` | 每端点持续时间（hey 模式） |
| `HEALTH_C` | `100` | `/health` 并发数 |
| `API_C` | `50` | API 端点并发数 |
| `PORT` | `8080` | 服务监听端口 |
| `BASE` | `http://127.0.0.1:8080` | 压测目标地址 |
| `START_SERVER` | `1` | 为 `1` 时自动启动 `examples/server` |
| `OUT` | — | 结果输出文件路径 |

工具链：优先 `hey`，否则使用系统 `ab`（约 5 万 / 2 万请求档）。

### 示例

```bash
# 服务已手动启动时
START_SERVER=0 ./scripts/stress_test.sh

# 缩短时长、保存日志
DURATION=15s OUT=stress.log ./scripts/stress_test.sh

# 完整套件并归档
BENCH_COUNT=3 OUT_DIR=./perf-out ./scripts/run_perf.sh
```

## 夹具维护

`requests.json` 中每个场景的主探针必须与对应 JSON 文件 `requests[0]` 一致
（`TestBenchProbeRequestsAlignWithCorpus`）。

各场景最少路由数（`TestBenchCorpusLoads`）：minimal ≥ 1、static ≥ 10、
parametric ≥ 5、middleware ≥ 10、large ≥ 100。