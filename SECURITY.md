# Security Policy

## 报告漏洞

如果你发现了 common 库中的安全漏洞，请**不要**通过公开的 GitHub Issue 报告。
请通过以下方式私下联系：

- 在 GitHub 上使用 [Report a vulnerability](https://github.com/452562082/common/security/advisories/new) 私密报告
- 或者邮件给仓库维护者

我们会在 **5 个工作日内**确认收到，并在评估后给出修复时间表。

## 支持的版本

主分支始终接受安全补丁。已发布的 tag 暂时不做单独的安全分支维护 ——
请使用 latest 主分支或最新 release。

## 内置安全特性

本库默认开启的安全防护包括：

| 包 | 默认安全行为 |
|---|---|
| `captcha` | 答案使用 `crypto/rand` + 拒绝采样去偏；常量时间比较；后台 GC |
| `websocket` | 默认拒绝跨源升级（防 CSWSH）；显式 `AllowedOrigins` 才允许 |
| `mail` | CRLF 头注入校验；凭证必须配 TLS |
| `httpclient` | 内置 `DenyPrivateIP`/`AllowHost` 防 SSRF；`MaxRedirects` 防重定向绕过；TLS 验证开启 |
| `httpserver` | 默认 4 MiB body size 限制；access log 不记 raw query；health endpoint 不泄漏内部错误 |
| `jwt` | 强校验 `alg` + `kid`，防 algorithm confusion |
| `toolbox` | pprof 文件 0600 权限（防泄漏 in-memory secrets） |
| `limiter` | `MaxKeys` + LRU 驱逐，防高基数 DoS |
| `distlock` | `crypto/rand` token + Lua 安全释放（compare-and-delete） |
| `nats`/`kafka`/`etcd` | 用户回调全部 panic-recover，防业务 panic 崩 dispatcher |

完整变更见 [CHANGELOG.md](./CHANGELOG.md)。

## CI 安全扫描

每次 push / PR 自动触发：

- `go vet ./...` — 静态检查
- `golangci-lint` (v2) — 包含 `gosec` 等安全规则
- `govulncheck ./...` — 扫描已知 stdlib + 依赖 CVE

## 依赖管理

`dependabot` 每周一扫描 `go.mod` 和 GitHub Actions 版本，自动开 PR
升级 minor / patch。
