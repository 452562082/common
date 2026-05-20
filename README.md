# common

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

An opinionated collection of Go building blocks for a Go service:
structured logging, HTTP client, SQL DB, config loader, graceful shutdown,
plus client wrappers for Kafka, Elasticsearch, ZooKeeper, and a generic
LRU cache.

All wrappers are `context`-aware, log via the standard-library `log/slog`, and
wrap underlying errors with `%w` so they're inspectable via `errors.Is` / `errors.As`.

## Usage

This is a **local-only** Go module — `module common` has no host prefix, so
`go get common` will not work. Consume it via one of:

### Option A — Go workspace (recommended, Go 1.18+)

From the directory that contains both this repo and your consuming project:

```bash
go work init
go work use ./common ./my-project
```

Then `import "common/kafka"` from `my-project` resolves directly.

### Option B — `replace` directive in your `go.mod`

```
require common v0.0.0-00010101000000-000000000000
replace common => /absolute/path/to/this/repo
```

Requires Go 1.25+.

## Packages

| Package | Description | Underlying client |
|---|---|---|
| **Application building blocks** | | |
| `logger` | `log/slog` wrapper: JSON/text, file rotation, ctx-propagated trace/request/user IDs | [`gopkg.in/natefinch/lumberjack.v2`](https://github.com/natefinch/lumberjack) |
| `httpclient` | `net/http` + timeouts, connection pool, retry with exponential backoff, JSON helpers | stdlib |
| `httpserver` | chi router + middleware (access log, recovery, request ID, CORS, metrics, tracing) | [`github.com/go-chi/chi/v5`](https://github.com/go-chi/chi) |
| `db` | `database/sql` + `sqlx` + `WithTx`, slow-query logging, unique-violation detector | [`github.com/jmoiron/sqlx`](https://github.com/jmoiron/sqlx) |
| `config` | Strongly-typed config loader: defaults → YAML → env, with nested structs | [`gopkg.in/yaml.v3`](https://gopkg.in/yaml.v3) |
| `graceful` | Long-running component orchestration: signal handling, reverse-order shutdown | stdlib |
| `validate` | Struct validation wrapper with JSON-tag-aware field names and friendly errors | [`go-playground/validator/v10`](https://github.com/go-playground/validator) |
| **Observability** | | |
| `metrics` | Prometheus client wrapper with namespaced `Counter` / `Gauge` / `Histogram` / `Timer` | [`prometheus/client_golang`](https://github.com/prometheus/client_golang) |
| `tracing` | OpenTelemetry SDK + OTLP/HTTP exporter; W3C propagator wired automatically | [`go.opentelemetry.io/otel`](https://opentelemetry.io) |
| **Concurrency & utilities** | | |
| `async` | Bounded `Pool` (worker queue) + `Retry` (exponential backoff with jitter) + `singleflight` re-export | `golang.org/x/sync` |
| `idgen` | UUID v4 / UUID v7 / Snowflake (int64, base58) | [`google/uuid`](https://github.com/google/uuid), [`bwmarrin/snowflake`](https://github.com/bwmarrin/snowflake) |
| `limiter` | Per-key token-bucket + sliding-window rate limiters (in-process) | `golang.org/x/time` |
| `breaker` | Circuit breaker with generic `Do` / `DoValue`, half-open recovery | [`sony/gobreaker/v2`](https://github.com/sony/gobreaker) |
| **Auth & storage** | | |
| `jwt` | JWT issue/verify with HS256 / RS256, typed `ErrExpired` / `ErrInvalid` | [`golang-jwt/jwt/v5`](https://github.com/golang-jwt/jwt) |
| `redis` | Unified Standalone / Cluster / Sentinel client + ping helper | [`redis/go-redis/v9`](https://github.com/redis/go-redis) |
| `mongo` | MongoDB client wrapper: Open / Ping / DB / Collection helpers | [`mongo-driver/v2`](https://github.com/mongodb/mongo-go-driver) |
| `distlock` | Redis-backed distributed lock with SET-NX + Lua release + optional auto-renew | `redis/go-redis/v9` |
| `objstore` | S3-compatible object store (AWS / MinIO / OSS / COS / R2) | [`aws-sdk-go-v2/service/s3`](https://github.com/aws/aws-sdk-go-v2) |
| **gRPC & jobs & errors & mail** | | |
| `grpcserver` | gRPC server + interceptor chain (logger / recovery / tracing) + health service | [`google.golang.org/grpc`](https://grpc.io) |
| `grpcclient` | Minimal gRPC client dial helper with tracing propagation | `google.golang.org/grpc` |
| `cron` | Cron scheduler (robfig/cron) with ctx-aware jobs + panic recovery | [`robfig/cron/v3`](https://github.com/robfig/cron) |
| `errpkg` | Typed errors: `Code` / `Status` / cause / stack + multi-error via `errors.Join` | stdlib |
| `mail` | SMTP sender with TLS / AUTH / multipart (text+HTML) + attachments | stdlib |
| **Coordination & realtime & misc** | | |
| `etcd` | etcd v3 client: KV / lease-based service registration / Mutex / Watch | [`go.etcd.io/etcd/client/v3`](https://etcd.io) |
| `nats` | NATS pub/sub + JetStream wrapper | [`nats.go`](https://github.com/nats-io/nats.go) |
| `websocket` | WebSocket with single-write goroutine + Hub broadcast + slow-consumer drop | [`gorilla/websocket`](https://github.com/gorilla/websocket) |
| `i18n` | Lightweight catalog-based translator with fallback + Accept-Language parsing | `golang.org/x/text` |
| `captcha` | Image PNG CAPTCHA with pluggable Store (single-use, TTL) | stdlib |
| **Infrastructure clients** | | |
| `kafka` | Sync / async producers and a consumer-group helper | [`github.com/IBM/sarama`](https://github.com/IBM/sarama) |
| `elastic` | Index/Get/Update/Delete/Search/Bulk/Scroll | [`github.com/elastic/go-elasticsearch/v8`](https://github.com/elastic/go-elasticsearch) |
| `zk` | ZooKeeper client (watch data/children) + server (auto-reregister) | [`github.com/go-zookeeper/zk`](https://github.com/go-zookeeper/zk) |
| `lru` | Generic in-memory LRU (with TTL) + Redis-backed LRU sharing the same interface | [`github.com/redis/go-redis/v9`](https://github.com/redis/go-redis) |
| **Helpers** | | |
| `env` | Read infrastructure endpoints from `,`/`;`-separated env vars | stdlib |
| `toolbox` | pprof / GC helpers for in-process diagnostics | stdlib |
| `utils` | Buffer pool, zero-copy `string` ⇄ `[]byte` helpers | stdlib |

## Quick examples

### Kafka — sync producer

```go
p, err := kafka.NewSyncProducer([]string{"localhost:9092"}, "events")
if err != nil { log.Fatal(err) }
defer p.Close()

_ = p.SendString("user-1", `{"action":"login"}`)
```

### Kafka — consumer group

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

### Elasticsearch — index + search

```go
es, _ := elastic.NewClient(elastic.Config{Addresses: []string{"http://localhost:9200"}})

_ = es.Index(ctx, "users", "u-1", map[string]any{"name": "Ada", "age": 36})

res, _ := es.Search(ctx, []string{"users"}, map[string]any{
    "query": map[string]any{"match": map[string]any{"name": "ada"}},
})
fmt.Println("total hits:", res.TotalHits)
```

### ZooKeeper — service registration

```go
srv, _ := zk.NewServer([]string{"localhost:2181"})
defer srv.Close()

// Ephemeral node under /services/billing/<host>
_ = srv.Register("/services/billing", "host.example:8080", []byte("ready"), true)
```

### LRU cache — in-memory (generic)

```go
c := lru.NewMemCache[string, *User](lru.MemOptions[string, *User]{
    Capacity: 10_000,
    TTL:      time.Hour,
    OnEvict:  func(k string, v *User) { metrics.IncEviction() },
})
_ = c.Set(ctx, "sid-1", user)
u, ok, _ := c.Get(ctx, "sid-1")
```

### LRU cache — Redis-backed (generic)

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

Both implement the same `Cache[K, V]` interface, so they're swappable at the call site.

## Development

```bash
make fmt vet test    # default checks
make cover           # coverage report -> coverage.html
make lint            # requires golangci-lint
```

## License

[MIT](LICENSE)
