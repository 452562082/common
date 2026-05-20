# common

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

一套面向 Go 后端服务的通用基建库：结构化日志、HTTP 客户端/服务端、SQL 数据库、配置加载、优雅退出，加上 Kafka、Elasticsearch、ZooKeeper、Redis、MongoDB、etcd、NATS、对象存储、JWT 鉴权、定时任务、WebSocket、国际化、验证码等一系列开箱即用的客户端封装。

所有封装均：

- 支持 `context.Context`，可超时/取消
- 使用标准库 `log/slog` 输出日志
- 错误通过 `%w` 包装，可经 `errors.Is` / `errors.As` 检查
- 自带回归测试，`go test -race ./...` 全绿

## 使用方式

这是个**本地模块**（`module common`，不带 host 前缀），`go get common` 拉不到。两种引用方式：

### 方式 A — Go workspace（推荐，Go 1.18+）

在同时包含本仓库和你的项目的目录下：

```bash
go work init
go work use ./common ./my-project
```

然后在 `my-project` 里就可以 `import "common/kafka"`。

### 方式 B — `replace` 指令

在你项目的 `go.mod` 里：

```
require common v0.0.0-00010101000000-000000000000
replace common => /absolute/path/to/this/repo
```

要求 Go 1.25+。

## 包列表

| 包 | 说明 | 底层依赖 |
|---|---|---|
| **应用骨架** | | |
| `logger` | `log/slog` 封装：JSON/文本格式、文件轮转、自动注入 trace/request/user ID | [`lumberjack.v2`](https://github.com/natefinch/lumberjack) |
| `httpclient` | `net/http` + 超时、连接池、指数退避重试、JSON 助手、SSRF 防护 | stdlib |
| `httpserver` | chi 路由 + middleware（access log、recovery、request ID、CORS、metrics、tracing）+ body size 限制 | [`go-chi/chi/v5`](https://github.com/go-chi/chi) |
| `grpcserver` | gRPC 服务端 + interceptor 链（logger / recovery / tracing）+ 健康检查 | [`google.golang.org/grpc`](https://grpc.io) |
| `grpcclient` | gRPC 客户端 Dial 助手 + 自动注入 W3C tracecontext | `google.golang.org/grpc` |
| `db` | `database/sql` + `sqlx` + 事务助手 + 慢查询日志 + unique 违反检测 | [`jmoiron/sqlx`](https://github.com/jmoiron/sqlx) |
| `config` | 强类型配置加载：默认值 → YAML → 环境变量，支持嵌套结构 | [`yaml.v3`](https://gopkg.in/yaml.v3) |
| `graceful` | 长生命周期组件编排：信号处理 + 反序关闭 + 超时兜底 | stdlib |
| `validate` | 结构体校验，JSON tag 字段名，友好错误信息 | [`validator/v10`](https://github.com/go-playground/validator) |
| `errpkg` | 类型化错误：Code / Status / cause / stack + `errors.Join` 多错合并 | stdlib |
| **可观测性** | | |
| `metrics` | Prometheus 客户端封装：Counter / Gauge / Histogram / Timer | [`prometheus/client_golang`](https://github.com/prometheus/client_golang) |
| `tracing` | OpenTelemetry SDK + OTLP/HTTP exporter，自动配置 W3C propagator | [`opentelemetry-go`](https://opentelemetry.io) |
| **并发 & 限流 & 弹性** | | |
| `async` | 有界 worker Pool + 指数退避 Retry + singleflight | `golang.org/x/sync` |
| `limiter` | 令牌桶 + 滑动窗口限流器，支持 `MaxKeys` LRU 防 DoS | `golang.org/x/time` |
| `breaker` | 熔断器，支持泛型 `Do[T]` + 半开恢复 | [`sony/gobreaker/v2`](https://github.com/sony/gobreaker) |
| `cron` | 定时任务（基于 robfig/cron）+ ctx 感知 + panic 恢复 | [`robfig/cron/v3`](https://github.com/robfig/cron) |
| **认证 & 存储** | | |
| `jwt` | JWT 签发/校验（HS256 + RS256），含 `KeySet` 支持 kid 多密钥轮换 | [`golang-jwt/jwt/v5`](https://github.com/golang-jwt/jwt) |
| `redis` | 统一 Standalone / Cluster / Sentinel 客户端 + Pinger 健康检查 | [`redis/go-redis/v9`](https://github.com/redis/go-redis) |
| `mongo` | MongoDB 客户端封装：Open / Ping / DB / Collection 助手 | [`mongo-driver/v2`](https://github.com/mongodb/mongo-go-driver) |
| `distlock` | 基于 Redis 的分布式锁（SET-NX + Lua 安全释放 + 自动续约） | `redis/go-redis/v9` |
| `objstore` | S3 兼容对象存储（AWS / MinIO / OSS / COS / R2） | [`aws-sdk-go-v2/service/s3`](https://github.com/aws/aws-sdk-go-v2) |
| **协调 & 实时 & 杂项** | | |
| `etcd` | etcd v3 客户端：KV / 基于 lease 的服务注册 / Mutex 分布式锁 / Watch | [`go.etcd.io/etcd/client/v3`](https://etcd.io) |
| `nats` | NATS pub/sub + JetStream 持久订阅 | [`nats.go`](https://github.com/nats-io/nats.go) |
| `websocket` | WebSocket：单写 goroutine、ping/pong 心跳、Hub 广播、慢客户端剔除 | [`gorilla/websocket`](https://github.com/gorilla/websocket) |
| `i18n` | 轻量多语言翻译器，支持 fallback + Accept-Language 解析 | `golang.org/x/text` |
| `captcha` | PNG 图形验证码（stdlib 绘制）+ 可插拔 Store + 后台 GC + 常量时间比较 | stdlib |
| `mail` | SMTP 发送（TLS + AUTH + multipart text/HTML + 附件 + CRLF 防注入） | stdlib |
| `idgen` | UUID v4 / UUID v7 / Snowflake（int64 / base58） | [`google/uuid`](https://github.com/google/uuid), [`bwmarrin/snowflake`](https://github.com/bwmarrin/snowflake) |
| **基础设施客户端** | | |
| `kafka` | 同步 / 异步生产者 + 消费者组（自动 panic recover） | [`IBM/sarama`](https://github.com/IBM/sarama) |
| `elastic` | Index / Get / Update / Delete / Search / Bulk / Scroll | [`go-elasticsearch/v8`](https://github.com/elastic/go-elasticsearch) |
| `zk` | ZooKeeper：客户端 watch（带 buffer + ctx 关闭）+ 服务端自动重注册 | [`go-zookeeper/zk`](https://github.com/go-zookeeper/zk) |
| `lru` | 泛型内存 LRU（带 TTL）+ Redis 后端 LRU，统一 `Cache[K, V]` 接口 | `redis/go-redis/v9` |
| **辅助** | | |
| `env` | 从 `,` / `;` 分隔的环境变量读取基础设施端点 | stdlib |
| `toolbox` | pprof / GC 诊断助手（profile 文件 0600 权限） | stdlib |
| `utils` | 字节缓冲池、零拷贝 `string` ⇄ `[]byte` 转换 | stdlib |

## 快速示例

### Kafka — 同步生产者

```go
p, err := kafka.NewSyncProducer([]string{"localhost:9092"}, "events")
if err != nil { log.Fatal(err) }
defer p.Close()

_ = p.SendString(ctx, "user-1", `{"action":"login"}`)
```

### Kafka — 消费者组

```go
g, _ := kafka.NewConsumerGroup([]string{"localhost:9092"}, []string{"events"}, "billing")
defer g.Close()

ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
defer cancel()

_ = g.Consume(ctx, func(ctx context.Context, msg *sarama.ConsumerMessage) error {
    fmt.Printf("%s/%d@%d: %s\n", msg.Topic, msg.Partition, msg.Offset, msg.Value)
    return nil
})
```

### Elasticsearch — 索引 + 搜索

```go
es, _ := elastic.NewClient(elastic.Config{Addresses: []string{"http://localhost:9200"}})

_ = es.Index(ctx, "users", "u-1", map[string]any{"name": "Ada", "age": 36})

res, _ := es.Search(ctx, []string{"users"}, map[string]any{
    "query": map[string]any{"match": map[string]any{"name": "ada"}},
})
fmt.Println("匹配条数:", res.TotalHits)
```

### ZooKeeper — 服务注册

```go
srv, _ := zk.NewServer([]string{"localhost:2181"})
defer srv.Close()

// 在 /services/billing/<host> 下创建临时节点
_ = srv.Register("/services/billing", "host.example:8080", []byte("ready"), true)
```

### LRU 缓存 — 内存（泛型）

```go
c := lru.NewMemCache[string, *User](lru.MemOptions[string, *User]{
    Capacity: 10_000,
    TTL:      time.Hour,
    OnEvict:  func(k string, v *User) { metrics.IncEviction() },
})
_ = c.Set(ctx, "sid-1", user)
u, ok, _ := c.Get(ctx, "sid-1")
```

### LRU 缓存 — Redis 后端（泛型）

```go
encKey, decKey := lru.StringKeyCodec()
rc, err := lru.NewRedisCache[string, *User](lru.RedisOptions[string, *User]{
    Name:        "sessions",
    Addr:        "localhost:6379",
    Capacity:    100_000,
    TTL:         time.Hour,
    EncodeKey:   encKey,
    DecodeKey:   decKey,
    EncodeValue: func(u *User) ([]byte, error) { return json.Marshal(u) },
    DecodeValue: func(b []byte) (*User, error) {
        var u User
        return &u, json.Unmarshal(b, &u)
    },
})
if err != nil { /* ... */ }
defer rc.Close()
_ = rc.Set(ctx, "sid-1", user)
```

两种实现都满足同一个 `Cache[K, V]` 接口，调用方可无缝替换。

### HTTP 客户端 — SSRF 防护

```go
c := httpclient.New(httpclient.Options{
    Timeout:      10 * time.Second,
    MaxRetries:   2,
    AllowHost:    httpclient.DenyPrivateIP, // 拒绝 RFC1918 / loopback / 云元数据
    MaxRedirects: 3,
})
err := c.GetJSON(ctx, userSuppliedURL, &out)
```

### JWT — Key 轮换

```go
ks := jwt.NewKeySet("billing-iss", time.Hour)
_ = ks.AddHMACKey("k1", []byte("old-secret"))
_ = ks.AddHMACKey("k2", []byte("new-secret"))
_ = ks.SetActive("k2") // 新签发的 token 使用 k2，老 token 仍可验证

token, _ := jwt.KeySetSign(ks, myClaims{
    RegisteredClaims: ks.NewClaims(),
    UID:              "u-1",
})
out, err := jwt.KeySetVerify[myClaims](ks, token)
```

### 优雅启动 + 关闭

```go
app := graceful.New(graceful.Options{ShutdownTimeout: 30 * time.Second})

app.Add("kafka-consumer",
    func(ctx context.Context) error { return consumer.Run(ctx) },
    consumer.Close,
)
app.Add("http-server",
    server.Start,
    server.Shutdown,
)

if err := app.Run(context.Background()); err != nil {
    log.Fatal(err)
}
```

更多可跑示例见 [`examples/`](./examples) 目录。

## 安全姿态

经过两轮安全审计与修复，本库默认安全配置包括：

- **captcha / distlock** 使用 `crypto/rand` 产生不可预测的答案/Token
- **websocket** 默认拒绝跨源升级（防 CSWSH），需显式配置 `AllowedOrigins`
- **mail** 拒绝 CRLF 头注入，凭证强制走 TLS
- **httpclient** 内置 `DenyPrivateIP` 防 SSRF，初始请求和重定向都校验
- **httpserver** 默认 4 MiB body size 限制，access log 不记录 raw query
- **jwt** 强校验 `alg` + `kid`，防 algorithm confusion 攻击
- **toolbox** pprof 文件 0600 权限（防泄漏 in-memory secrets）
- **limiter** `MaxKeys` + LRU 驱逐，防高基数 DoS
- **captcha / nats / kafka / etcd** 回调全部 panic-recover
- **health endpoint** 不暴露内部错误细节
- **db.IsUniqueViolation** typed 错误检测（pgx SQLState + MySQL Number）

完整变更见 [CHANGELOG.md](./CHANGELOG.md)。

## 开发

```bash
make fmt vet test    # 格式化 / 静态检查 / 测试
make cover           # 覆盖率报告 → coverage.html
make lint            # 需安装 golangci-lint
```

## License

[MIT](LICENSE)
