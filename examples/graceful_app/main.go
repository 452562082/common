// Minimal application skeleton showing how the library's building blocks
// compose: config → logger → httpserver → graceful.
//
//	go run ./examples/graceful_app
//
// In another terminal:
//
//	curl http://localhost:8080/hello
//	curl http://localhost:8080/healthz
//
// Ctrl-C triggers graceful shutdown — every component is stopped in reverse
// registration order, bounded by Options.ShutdownTimeout.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"common/config"
	"common/graceful"
	"common/httpserver"
	"common/logger"
)

// Cfg is the strongly-typed config struct. Pass via APP_* env vars or
// supply a YAML file path through APP_CONFIG.
type Cfg struct {
	Listen   string `yaml:"listen"    env:"LISTEN"    default:":8080"`
	LogLevel string `yaml:"log_level" env:"LOG_LEVEL" default:"info"`
}

func main() {
	// 1. Load config (defaults → optional YAML → env-var overrides).
	var cfg Cfg
	if err := config.Load(&cfg, config.Options{
		File:         os.Getenv("APP_CONFIG"),
		FileOptional: true,
		EnvPrefix:    "APP",
	}); err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}

	// 2. Logger — JSON to stderr, level from config.
	log := logger.New(logger.Options{
		Level:  logger.ParseLevel(cfg.LogLevel),
		Format: logger.FormatJSON,
	})
	logger.SetDefault(log)
	logger.SetAsStdDefault(log)

	// 3. HTTP server — chi router with default middleware stack.
	srv := httpserver.New(httpserver.Options{
		Addr:   cfg.Listen,
		Logger: log,
		HealthCheck: func(context.Context) error {
			// In a real app: ping DB, redis, downstreams, ...
			return nil
		},
	})
	srv.Router().Get("/hello", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello\n"))
	})

	// 4. Background worker — typical scheduled / long-running task.
	worker := &demoWorker{log: log}

	// 5. graceful — wires SIGINT/SIGTERM + reverse-order shutdown.
	app := graceful.New(graceful.Options{
		Logger:          log,
		ShutdownTimeout: 30 * time.Second,
	})
	app.Add("http-server", srv.Start, srv.Shutdown)
	app.Add("demo-worker", worker.Run, worker.Stop)

	if err := app.Run(context.Background()); err != nil {
		log.Error("app exited with error", "err", err)
		os.Exit(1)
	}
}

// demoWorker is a stand-in for any context-driven background loop.
type demoWorker struct {
	log *slog.Logger
}

func (w *demoWorker) Run(ctx context.Context) error {
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			w.log.InfoContext(ctx, "demo-worker tick")
		}
	}
}

func (w *demoWorker) Stop(context.Context) error {
	w.log.Info("demo-worker stopping")
	return nil
}
