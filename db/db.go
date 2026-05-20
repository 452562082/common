// Package db is a thin wrapper around database/sql + sqlx.
//
// It adds:
//
//   - Driver-agnostic Open with sensible pool defaults.
//   - A WithTx helper that handles begin/commit/rollback correctly across
//     panics and context cancellation.
//   - Optional slow-query logging via slog.
//
// Supported drivers (register the one you need by blank-importing it):
//
//	import _ "github.com/go-sql-driver/mysql"     // db.DriverMySQL
//	import _ "github.com/jackc/pgx/v5/stdlib"     // db.DriverPostgres
//	import _ "modernc.org/sqlite"                 // db.DriverSQLite
package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Driver identifies the database flavour. Use these constants when populating
// Options.Driver — they map to the strings expected by sql.Open.
type Driver string

const (
	DriverMySQL    Driver = "mysql"
	DriverPostgres Driver = "pgx"
	DriverSQLite   Driver = "sqlite"
)

// Options configures Open.
type Options struct {
	Driver Driver // required
	DSN    string // required

	// Connection-pool tuning. Zero falls back to sensible defaults.
	MaxOpenConns    int           // default 25
	MaxIdleConns    int           // default 25
	ConnMaxLifetime time.Duration // default 30m
	ConnMaxIdleTime time.Duration // default 10m

	// PingTimeout caps the initial connectivity check. Default 5s.
	// Use 0 to disable the check.
	PingTimeout time.Duration

	// SlowThreshold logs queries that take longer than this at WARN level.
	// Zero disables slow-query logging.
	SlowThreshold time.Duration

	// Logger is used for slow-query logging. nil falls back to slog.Default().
	Logger *slog.Logger
}

// DB wraps *sqlx.DB and provides ctx-friendly helpers.
type DB struct {
	*sqlx.DB
	opts Options
	log  *slog.Logger
}

// Open dials the database, configures the pool, and verifies connectivity.
func Open(ctx context.Context, opts Options) (*DB, error) {
	if opts.Driver == "" {
		return nil, errors.New("db: Driver is required")
	}
	if opts.DSN == "" {
		return nil, errors.New("db: DSN is required")
	}
	applyDBDefaults(&opts)

	x, err := sqlx.Open(string(opts.Driver), opts.DSN)
	if err != nil {
		return nil, fmt.Errorf("db: open %s: %w", opts.Driver, err)
	}
	x.SetMaxOpenConns(opts.MaxOpenConns)
	x.SetMaxIdleConns(opts.MaxIdleConns)
	x.SetConnMaxLifetime(opts.ConnMaxLifetime)
	x.SetConnMaxIdleTime(opts.ConnMaxIdleTime)

	if opts.PingTimeout > 0 {
		pingCtx, cancel := context.WithTimeout(ctx, opts.PingTimeout)
		defer cancel()
		if err := x.PingContext(pingCtx); err != nil {
			_ = x.Close()
			return nil, fmt.Errorf("db: ping: %w", err)
		}
	}

	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	return &DB{DB: x, opts: opts, log: log}, nil
}

func applyDBDefaults(o *Options) {
	if o.MaxOpenConns == 0 {
		o.MaxOpenConns = 25
	}
	if o.MaxIdleConns == 0 {
		o.MaxIdleConns = 25
	}
	if o.ConnMaxLifetime == 0 {
		o.ConnMaxLifetime = 30 * time.Minute
	}
	if o.ConnMaxIdleTime == 0 {
		o.ConnMaxIdleTime = 10 * time.Minute
	}
	if o.PingTimeout == 0 {
		o.PingTimeout = 5 * time.Second
	}
}

// Close releases the connection pool.
func (db *DB) Close() error {
	if err := db.DB.Close(); err != nil {
		return fmt.Errorf("db: close: %w", err)
	}
	return nil
}

// ExecContext wraps sqlx's ExecContext with slow-query logging.
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	res, err := db.DB.ExecContext(ctx, query, args...)
	db.logSlow(ctx, "exec", query, args, start, err)
	if err != nil {
		return nil, fmt.Errorf("db: exec: %w", err)
	}
	return res, nil
}

// QueryRowxContext wraps sqlx with slow-query logging.
func (db *DB) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	start := time.Now()
	row := db.DB.QueryRowxContext(ctx, query, args...)
	db.logSlow(ctx, "queryRow", query, args, start, nil)
	return row
}

// QueryxContext wraps sqlx with slow-query logging.
func (db *DB) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	start := time.Now()
	rows, err := db.DB.QueryxContext(ctx, query, args...)
	db.logSlow(ctx, "query", query, args, start, err)
	if err != nil {
		return nil, fmt.Errorf("db: query: %w", err)
	}
	return rows, nil
}

// SelectContext is sqlx.SelectContext with slow-query logging.
func (db *DB) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	start := time.Now()
	err := db.DB.SelectContext(ctx, dest, query, args...)
	db.logSlow(ctx, "select", query, args, start, err)
	if err != nil {
		return fmt.Errorf("db: select: %w", err)
	}
	return nil
}

// GetContext is sqlx.GetContext with slow-query logging.
func (db *DB) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	start := time.Now()
	err := db.DB.GetContext(ctx, dest, query, args...)
	db.logSlow(ctx, "get", query, args, start, err)
	if err != nil {
		return fmt.Errorf("db: get: %w", err)
	}
	return nil
}

func (db *DB) logSlow(ctx context.Context, op, query string, args []any, start time.Time, err error) {
	if db.opts.SlowThreshold <= 0 {
		return
	}
	elapsed := time.Since(start)
	if elapsed < db.opts.SlowThreshold {
		return
	}
	attrs := []any{
		slog.String("op", op),
		slog.Duration("elapsed", elapsed),
		slog.String("query", truncate(query, 256)),
		slog.Int("args", len(args)),
	}
	if err != nil {
		attrs = append(attrs, slog.Any("err", err))
	}
	db.log.LogAttrs(ctx, slog.LevelWarn, "db: slow query", toAttrs(attrs)...)
}

func toAttrs(in []any) []slog.Attr {
	out := make([]slog.Attr, 0, len(in))
	for _, v := range in {
		if a, ok := v.(slog.Attr); ok {
			out = append(out, a)
		}
	}
	return out
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// WithTx runs fn inside a transaction. Commit happens on nil error; otherwise
// rollback. If fn panics, the transaction is rolled back and the panic re-raised.
//
//	err := db.WithTx(ctx, nil, func(tx *sqlx.Tx) error {
//	    if _, err := tx.ExecContext(ctx, "..."); err != nil {
//	        return err
//	    }
//	    return nil
//	})
func (db *DB) WithTx(ctx context.Context, opts *sql.TxOptions, fn func(*sqlx.Tx) error) (err error) {
	tx, err := db.BeginTxx(ctx, opts)
	if err != nil {
		return fmt.Errorf("db: begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()
	if err = fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && !errors.Is(rbErr, sql.ErrTxDone) {
			return fmt.Errorf("db: rollback (after %v): %w", err, rbErr)
		}
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("db: commit: %w", err)
	}
	return nil
}

// IsUniqueViolation reports whether err is a unique-constraint violation
// (MySQL 1062, Postgres SQLSTATE 23505, SQLite "UNIQUE constraint failed").
//
// Detection order:
//  1. Driver-specific error types via reflection, so we don't force callers
//     to import a driver they don't use. This catches `*mysql.MySQLError`
//     (field Number == 1062) and any error exposing `SQLState() string`
//     returning "23505" (pgx, lib/pq, etc).
//  2. Error-message substring as a fallback.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// Path 1a: pgx / lib/pq expose SQLState() string.
	type sqlStater interface{ SQLState() string }
	var ss sqlStater
	if errors.As(err, &ss) && ss.SQLState() == "23505" {
		return true
	}
	// Path 1b: MySQL driver's *mysql.MySQLError has a `Number uint16` field.
	if isMySQLDupKey(err) {
		return true
	}
	// Path 2: string match. Robust for sqlite, gorm wrappers, and exotic
	// drivers that don't expose typed errors.
	msg := err.Error()
	return strings.Contains(msg, "Error 1062") ||
		strings.Contains(msg, "duplicate key value") ||
		strings.Contains(msg, "SQLSTATE 23505") ||
		strings.Contains(msg, "UNIQUE constraint failed")
}

// isMySQLDupKey reflects through the error chain looking for a type with a
// uint16 field named Number == 1062. We use reflection because importing the
// MySQL driver just for one type check would force every caller of this
// package to drag in the driver.
func isMySQLDupKey(err error) bool {
	for cur := err; cur != nil; cur = errors.Unwrap(cur) {
		v := reflect.ValueOf(cur)
		if v.Kind() == reflect.Pointer {
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			continue
		}
		f := v.FieldByName("Number")
		if !f.IsValid() || f.Kind() != reflect.Uint16 {
			continue
		}
		if f.Uint() == 1062 {
			return true
		}
	}
	return false
}
