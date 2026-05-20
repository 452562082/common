# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed (second renovation pass)

- `hbase` package — HBase usage has declined sharply; if you need it, depend on
  `github.com/tsuna/gohbase` directly.
- `hdfs` package — on-prem HDFS is increasingly replaced by S3-compatible
  object storage; if you need it, depend on `github.com/colinmarc/hdfs/v2`
  directly.

### Changed (second renovation pass)

- `lru.Cache` is now generic: `Cache[K comparable, V any]`. The old
  `MemCache`/`RedisCache` types and their `Add/Modify/RemoveAll/...` interface
  have been replaced with `Get / Set / Delete / Len / Range / Clear /
  PurgeExpired / Close`.
- `lru.RedisCache` now accepts `context.Context` on every method (previously
  all calls used `context.Background()`).
- `lru.RedisCache` no longer maintains an unbounded in-process "memcache"
  layer that bypassed LRU updates.
- `zk.Client` channels are buffered and use a conflate-on-overflow strategy;
  `Close()` waits for watcher goroutines to exit and is safe to call multiple
  times.
- `kafka.ConsumerGroup.Consume` now distinguishes context cancellation from
  fatal errors. `ConsumerGroupOptions` exposes `Version` / `InitialOffset` /
  `LogErrors`.
- `kafka.SyncProducer.Send*` methods now take `context.Context` (soft cancel).
- `elastic.Client.Index` defaults to `refresh=false`; new `IndexWithRefresh`
  helper accepts the explicit policy. Constants `RefreshFalse / RefreshTrue /
  RefreshWaitFor` exported.
- `env` host getters accept both `,` and `;` separators (previously `;` only).
  New `RedisHosts()` getter.
- `utils.S2B` / `B2S` deprecated in favour of `StringToBytes` /
  `BytesToString` (the old names remain as wrappers).
- `toolbox` no longer calls `log.Fatal`; every helper returns an error.
  `GetCPUProfile` takes a context and a duration.
- Go toolchain bumped to 1.25; CI matrix is 1.24 / 1.25.

### Security (ninth pass — second-round audit)

#### Fixed

- **nats / kafka / etcd**: user message / event callbacks now run under a
  panic recover. A buggy handler used to take down the underlying client's
  internal goroutine — now it's logged and the dispatcher keeps running.
- **captcha.MemoryStore**: added background sweeper (default 1-minute
  cadence) that evicts expired entries. Without it, abandoned captchas
  piled up indefinitely — an unauthenticated DoS vector since captcha
  endpoints are public by design. New `NewMemoryStoreWithInterval` + `Close`.
- **limiter**: `TokenBucket.MaxKeys` + `SlidingWindow.SetMaxKeys` cap the
  number of distinct keys held in memory, with LRU eviction. Prevents
  cardinality DoS when the key derives from untrusted input (IP, header).
- **httpserver**: health endpoint no longer returns the raw error message.
  Failures are logged via slog server-side; clients see only
  `{"status":"unhealthy"}`. Internal infrastructure details (DB host:port,
  cluster topology) stay private.
- **captcha**: `Verify` now uses `crypto/subtle.ConstantTimeCompare` —
  closes the (admittedly tiny) timing side-channel on answer comparison.
- **logger**: package doc carries a prominent "do not log secrets"
  warning. slog records attributes verbatim, so callers have to be careful
  what they pass.

### Security (eighth pass — audit + hardening)

#### Fixed

- **captcha**: answer generation now uses `crypto/rand` with rejection
  sampling (was `math/rand` — predictable PRNG, recoverable from a few
  observed answers).
- **websocket**: default `CheckOrigin` now rejects browser cross-site
  upgrades; added `Options.AllowedOrigins []string` + `"*"` wildcard
  (was unconditionally allowing all origins — CSWSH risk).
- **mail**: `Message.validate()` rejects CR / LF in From / To / CC / BCC,
  custom headers, and attachment filenames (prevents SMTP header injection).
- **mail**: `NewSender` refuses Username / Password without TLS (prevents
  PLAIN AUTH leakage over plaintext).
- **toolbox**: heap / CPU profiles now written mode 0600 (was 0666 —
  profiles routinely contain in-memory secrets).
- **httpclient**: new `AllowHost(host) bool` hook + `DenyPrivateIP`
  helper (rejects RFC1918, loopback, link-local, multicast — blocks SSRF
  to cloud metadata services and internal networks). Applies to the initial
  request AND every redirect.
- **httpclient**: new `MaxRedirects` option (`-1` disables redirects
  entirely; prevents the public-→-private redirect SSRF bypass).
- **httpserver**: new `Options.MaxBodyBytes` (default 4 MiB) + `MaxBody`
  middleware — bounds memory use under hostile POSTs.
- **httpserver/middleware**: `AccessLog` no longer logs the raw query
  string; new `AccessLogWithOptions` accepts a `LogQueryKeys` allowlist,
  redacting everything else as `***`.
- **jwt**: new `KeySet` type for safe key rotation. Tokens carry a `kid`
  header; verify picks the key by kid. Missing / unknown kid is rejected.
  Supports HMAC + RSA + verify-only RSA (consumer side).
- **db**: `IsUniqueViolation` now does typed detection first (pgx /
  lib-pq via `SQLState()`, MySQL via reflection on the `Number` field),
  falling back to string matching. No new required dependencies.

### Added (seventh pass — coordination + realtime + misc)

- `etcd` package — clientv3 wrapper: `Get/Put/Delete/List` KV helpers,
  `Register` (lease-based ephemeral service entry with keepalive),
  `WatchPrefix`, `Lock`/`TryLock` via `concurrency.Mutex`.
- `nats` package — `Connect` + core pub/sub (`Publish` / `Subscribe` /
  `QueueSubscribe` / `Request`) + lazy JetStream handle.
- `websocket` package — gorilla/websocket wrapper with per-conn write
  goroutine (single-writer rule satisfied for free), built-in ping/pong,
  bounded send queue, `Hub` broadcast that drops slow consumers.
- `i18n` package — multi-language catalogs keyed by `language.Tag`, JSON
  file loader, `Accept-Language`-aware Localizer, `{{name}}` placeholder
  substitution. Default-language fallback for missing keys.
- `captcha` package — PNG image CAPTCHA generator with stdlib-only drawing,
  single-use semantics, pluggable `Store` (`MemoryStore` included, Redis
  store left to the caller).

### Added (sixth pass — gRPC, jobs, errors, mail)

- `errpkg` package — typed `Error` carrying `Code` / `Status` / cause /
  captured stack; `errors.Is`/`As`-compatible; `Multi` accumulator built on
  `errors.Join`; `CodeOf` / `StatusOf` helpers.
- `grpcserver` package — gRPC server with chained unary/stream interceptors
  (recover / logger / tracing), built-in health service, optional reflection,
  `Start(ctx) / Shutdown(ctx)` lifecycle that plugs into `graceful.App`.
- `grpcclient` package — `Dial(ctx, target, Options)` helper with W3C
  tracecontext propagation and TLS / insecure toggle.
- `cron` package — `robfig/cron/v3` wrapper: ctx-aware `JobFunc`, panic
  recovery, slog log on every invocation, `Start(ctx) / Stop(ctx)` lifecycle.
- `mongo` package — `mongo-driver/v2` wrapper: `Open` with ping-on-open,
  default DB pin, `IsDuplicateKey` / `IsNoDocuments` helpers.
- `mail` package — net/smtp-based sender: STARTTLS / implicit TLS,
  PLAIN auth, multipart (text+HTML) bodies, attachments, ctx-aware `Send`.

### Added (fifth pass — auth, storage, resilience)

- `redis` package — Unified `Standalone / Cluster / Sentinel` go-redis
  wrapper with explicit ping-on-open, `Pinger` interface for health checks,
  `IsNil` helper.
- `distlock` package — Redis-backed distributed lock: SET-NX-PX acquire,
  Lua compare-and-delete release, optional auto-refresh goroutine, typed
  `ErrNotAcquired` / `ErrNotHeld`.
- `objstore` package — S3-compatible storage client wrapping
  `aws-sdk-go-v2/service/s3`. Put / Get / Head / Delete / List / Presign,
  works against AWS S3, MinIO, OSS, COS, R2, ...
- `jwt` package — Generic Sign / Verify on top of golang-jwt/v5, HS256 and
  RS256, `ErrExpired` / `ErrInvalid` for clean errors.Is matching.
- `limiter` package — Per-key in-process rate limiters: `TokenBucket` (wraps
  `golang.org/x/time/rate` with idle-eviction) + `SlidingWindow`.
- `breaker` package — Circuit breaker around `sony/gobreaker/v2` with
  generic `Do[T any]`, `OnStateChange` hook, exported `ErrOpen` /
  `ErrTooManyRequests`.

### Added (fourth pass — observability + concurrency + HTTP server)

- `metrics` package — Prometheus wrapper with `Counter` / `Gauge` /
  `Histogram` / `Summary` / `Timer` helpers, namespaced registries, default
  Go-runtime and process collectors, isolated `Handler()` for `/metrics`.
- `tracing` package — OpenTelemetry SDK initialiser: OTLP/HTTP exporter,
  W3C tracecontext + baggage propagator, sample-ratio sampler, helpers to
  pull trace/span IDs back out of a context.
- `httpserver` package — chi-based server with sensible timeouts, optional
  metrics and health endpoints, and a `Start(ctx) / Shutdown(ctx)` lifecycle
  that plugs directly into `graceful.App.Add`.
- `httpserver/middleware` sub-package — `AccessLog`, `Recover`, `CORS`,
  `Tracing` (auto-attaches the trace ID to `logger` ctx), `PrometheusMetrics`.
- `idgen` package — `UUID` (v4), `UUIDv7` (time-ordered), `Snowflake`
  (int64 / decimal / base58).
- `async` package — `Pool` (bounded worker pool), `Retry` (exponential
  backoff with jitter, `ShouldRetry` hook), `Group` (singleflight re-export).
- `validate` package — `go-playground/validator/v10` wrapper with
  JSON-tag-aware field names, structured `Errors` type for HTTP handlers,
  and `RegisterRule` for custom checks.

### Added (third pass — application building blocks)

- `logger` package — `log/slog` wrapper with JSON/text format, file rotation
  (lumberjack), and a context-aware handler that auto-attaches `trace_id` /
  `request_id` / `user_id` from `WithTraceID` / `WithRequestID` / `WithUserID`.
- `httpclient` package — `net/http` + timeouts, tuned connection pool,
  exponential-backoff retries with jitter, `GetJSON` / `PostJSON` helpers,
  rich `HTTPError` for >= 400 responses.
- `db` package — `database/sql` + `jmoiron/sqlx` wrapper with pool defaults,
  ping-on-open, slow-query logging via `slog`, `WithTx` (panic-safe
  begin/commit/rollback), `IsUniqueViolation` cross-driver helper.
- `config` package — strongly-typed loader: defaults → YAML → env with nested
  structs, `envPrefix` propagation, `[]string` from `,`/`;` lists.
- `graceful` package — orchestrates long-running components: starts them all,
  waits on signal / ctx / component-exit, then shuts them down in reverse
  order with a bounded timeout.

### Added (second renovation pass)

- Unit tests for `env`, `lru.MemCache`, `utils`, `toolbox`, `zk.parentPaths`.
- `examples/` directory with minimal runnable demos per package.
- `.gitignore` ignores `go.work` / `go.work.sum`.

## [Pre-removal renovation]

### Breaking — full renovation pass

This release rewrites the entire library. Every public name and import path has
changed. Treat it as a fresh start; there is no compatibility path from the
pre-`go.mod` codebase.

#### Added

- `go.mod` (module path: `common`, Go 1.23+).
- MIT `LICENSE`.
- `Makefile` (`fmt`, `vet`, `lint`, `test`, `cover`, `tidy`).
- `.golangci.yml` and `.github/workflows/ci.yml` for matrix CI + lint.
- `elastic` package: new generic v8 wrapper (Index / Get / Update / Delete /
  Search / Bulk / Scroll / CreateIndex / DeleteIndex / IndexExists).
- `hbase` package: rewritten on top of `tsuna/gohbase` (native Go HBase client,
  no Thrift required).
- `kafka.ConsumerGroup`: replaces the dead `wvanbergen/kafka` ZooKeeper-based
  consumer group with sarama's native Kafka-coordinated group.
- `hdfs.CheckHDFSAlive` now ctx-aware.
- `env.ErrUnset` sentinel error for `errors.Is` matching.

#### Changed

- All clients now accept `context.Context` as the first parameter (where the
  underlying library supports it).
- All errors are wrapped with `%w` and prefixed by package name.
- Logging routed through stdlib `log/slog` (no more custom Beego-style logger).
- Receivers renamed from `this`/`self` to short Go-idiomatic names.
- `lru.NewCacheMem` → `lru.NewMemCache`; `lru.NewCacheRedis` → `lru.NewRedisCache`
  (now takes a `RedisCacheOptions` struct).
- `kafka.NewKafkaSyncProducer` → `kafka.NewSyncProducer`.
- `kafka.NewKafkaAsyncProducer` → `kafka.NewAsyncProducer`.
- `zk.NewGozkClient` → `zk.NewClient`; `zk.NewGozkServer` → `zk.NewServer`.
- `hdfs.NewHdfsClient` / `NewHdfsClient2` → `hdfs.NewClient`.
- `hbase.HBaseHelper` → `hbase.Client`.

#### Removed

- `pamodels/` package (business-specific PingAn voice-print models).
- `elastic/goelastic*.go`, `elastic/estransfer/` (ASV-specific code with
  hard-coded business mappings).
- `utils/log/`, `utils/httplib/`, `utils/json/`, `utils/daemon/` (stdlib copies
  and Beego ports — use `log/slog`, `net/http`, `encoding/json` directly).
- `hbase/t_h_base_service-remote/` (Thrift CLI; the new client doesn't need it).
- `kafka/consumer_cluster.go` (`bsm/sarama-cluster` is dead — superseded by
  `sarama.ConsumerGroup`).
- All hard-coded internal IPs and business identifiers (`KST`,
  `kuaishangtong/...`, `asvserver/...`, `pingan/...`).

#### Dependency upgrades

| Was | Now |
|---|---|
| `git.apache.org/thrift.git` | (dropped — replaced by native client) |
| `github.com/Shopify/sarama` | `github.com/IBM/sarama` |
| `gopkg.in/olivere/elastic.v3` | `github.com/elastic/go-elasticsearch/v8` |
| `github.com/samuel/go-zookeeper` | `github.com/go-zookeeper/zk` |
| `github.com/go-redis/redis` | `github.com/redis/go-redis/v9` |
| `github.com/colinmarc/hdfs` | `github.com/colinmarc/hdfs/v2` |
| `github.com/wvanbergen/kafka/consumergroup` | `sarama.ConsumerGroup` (built-in) |
| `github.com/bsm/sarama-cluster` | `sarama.ConsumerGroup` (built-in) |
| `github.com/opentracing/opentracing-go` | (dropped — migrate to OpenTelemetry per call site) |
| `goal/github.com/coocood/freecache` | (dropped — stdlib `container/list` + `map` is sufficient) |
