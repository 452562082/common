# 贡献指南

感谢你考虑给 common 库贡献代码 / 文档 / bug 报告。

## 开发环境

- Go 1.25+
- 推荐安装 [golangci-lint](https://golangci-lint.run/) v2.x 用于本地 lint
- 推荐安装 [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) 扫漏洞

```bash
make fmt vet test    # 格式化 / vet / 测试
make cover           # 覆盖率报告 → coverage.html
make lint            # 需 golangci-lint
```

## 提 PR 前请确认

1. **测试通过**：`go test -race -count=1 ./...` 全绿
2. **lint 通过**：`make lint` 无报错
3. **godoc 完整**：新增的公开符号有文档注释
4. **errors.Is 友好**：错误用 `%w` 包装；新增的 sentinel error 在包文档里列出
5. **ctx-aware**：所有可能阻塞 / 网络 IO 的函数首参 `context.Context`
6. **panic 兜底**：新增的用户回调入口要有 `recover`
7. **不暴露 secret**：日志 / 错误 / health endpoint 不要包含内部 host:port、token

## Commit message 风格

- 一句话总结（< 70 字）
- 必要时空一行 + 详细说明
- 中文 / 英文均可
- 建议前缀：`feat:` / `fix:` / `sec:` / `docs:` / `refactor:` / `test:` / `ci:` / `deps:`

例：

```
fix: httpclient race in retry backoff jitter

- *rand.Rand 实例不是 goroutine-safe，多 goroutine 共享一个 Client
  时会 race
- 换 math/rand/v2（包级 API 自带并发安全），删除 Client.rng 字段
- 新增 TestRetry_ConcurrentBackoffIsRaceFree 压测
```

## 包结构约定

新增一个包时：

```
mypkg/
├── doc.go          # 仅含 // Package mypkg ... 文档
├── mypkg.go        # 核心类型 + 构造函数
├── mypkg_test.go   # 单测
├── example_test.go # 可选：pkg.go.dev 友好的 Example_xxx
└── bench_test.go   # 可选：性能基线 BenchmarkXxx
```

构造函数命名：

- `Open(ctx, opts)` — 涉及外部资源 + 健康检查的客户端（db / redis / mongo / etcd / objstore / nats）
- `New(opts)` — 纯内存类型 / 配置好的 builder（lru / breaker / limiter / cron / captcha / ...）

API 一致性：

- `Close() error` — 跟 `io.Closer` 对齐，即使内部从不出错也返回 nil
- 所有阻塞 / IO 函数首参 `context.Context`
- 错误 `%w` 包装 + 类型化 sentinel（`errors.Is/As` 友好）

## 报告漏洞

请走 [SECURITY.md](./SECURITY.md) 的私密渠道，不要开公开 Issue。

## License

提交的代码默认按 [MIT](./LICENSE) 授权。
