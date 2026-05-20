package db

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "test.db")
	d, err := Open(context.Background(), Options{
		Driver: DriverSQLite,
		DSN:    dsn,
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	return d
}

func TestOpenAndPing(t *testing.T) {
	d := openTestDB(t)
	if err := d.PingContext(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestOpen_MissingDriver(t *testing.T) {
	_, err := Open(context.Background(), Options{DSN: "x"})
	if err == nil {
		t.Fatal("expected error for missing driver")
	}
}

func TestOpen_MissingDSN(t *testing.T) {
	_, err := Open(context.Background(), Options{Driver: DriverSQLite})
	if err == nil {
		t.Fatal("expected error for missing DSN")
	}
}

func TestExecAndQuery(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()
	_, err := d.ExecContext(ctx, `CREATE TABLE users(id INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE)`)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := d.ExecContext(ctx, `INSERT INTO users(name) VALUES (?)`, "ada"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var name string
	if err := d.GetContext(ctx, &name, `SELECT name FROM users WHERE id = ?`, 1); err != nil {
		t.Fatalf("get: %v", err)
	}
	if name != "ada" {
		t.Errorf("name = %q", name)
	}

	var names []string
	if err := d.SelectContext(ctx, &names, `SELECT name FROM users ORDER BY id`); err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(names) != 1 || names[0] != "ada" {
		t.Errorf("got %v", names)
	}
}

func TestWithTx_Commit(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()
	_, _ = d.ExecContext(ctx, `CREATE TABLE t(v INTEGER)`)

	err := d.WithTx(ctx, nil, func(tx *sqlx.Tx) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO t VALUES (1), (2)`); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTx: %v", err)
	}

	var n int
	if err := d.GetContext(ctx, &n, `SELECT COUNT(*) FROM t`); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("rows committed = %d, want 2", n)
	}
}

func TestWithTx_Rollback(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()
	_, _ = d.ExecContext(ctx, `CREATE TABLE t(v INTEGER)`)

	sentinel := errors.New("boom")
	err := d.WithTx(ctx, nil, func(tx *sqlx.Tx) error {
		_, _ = tx.ExecContext(ctx, `INSERT INTO t VALUES (1)`)
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel, got %v", err)
	}

	var n int
	if err := d.GetContext(ctx, &n, `SELECT COUNT(*) FROM t`); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("expected rollback, got %d rows", n)
	}
}

func TestWithTx_PanicRollback(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()
	_, _ = d.ExecContext(ctx, `CREATE TABLE t(v INTEGER)`)

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic to propagate")
			}
		}()
		_ = d.WithTx(ctx, nil, func(tx *sqlx.Tx) error {
			_, _ = tx.ExecContext(ctx, `INSERT INTO t VALUES (1)`)
			panic("kaboom")
		})
	}()

	var n int
	_ = d.GetContext(ctx, &n, `SELECT COUNT(*) FROM t`)
	if n != 0 {
		t.Errorf("panic should rollback, got %d rows", n)
	}
}

func TestIsUniqueViolation(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()
	_, _ = d.ExecContext(ctx, `CREATE TABLE u(name TEXT UNIQUE)`)
	_, _ = d.ExecContext(ctx, `INSERT INTO u VALUES ('x')`)

	_, err := d.ExecContext(ctx, `INSERT INTO u VALUES ('x')`)
	if err == nil {
		t.Fatal("expected violation")
	}
	if !IsUniqueViolation(err) {
		t.Errorf("IsUniqueViolation didn't recognise: %v", err)
	}
}

// fakePgError mimics pgx/lib-pq style errors that expose SQLState().
type fakePgError struct{ code string }

func (e *fakePgError) Error() string       { return "pg error " + e.code }
func (e *fakePgError) SQLState() string    { return e.code }

// fakeMySQLError mimics *go-sql-driver/mysql.MySQLError with a uint16 Number.
type fakeMySQLError struct {
	Number  uint16
	Message string
}

func (e *fakeMySQLError) Error() string { return e.Message }

func TestIsUniqueViolation_TypedDetection(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"pg 23505 unique violation", &fakePgError{code: "23505"}, true},
		{"pg 23503 fk violation", &fakePgError{code: "23503"}, false},
		{"mysql 1062 dup key", &fakeMySQLError{Number: 1062, Message: "Error 1062: Duplicate entry"}, true},
		{"mysql 1452 fk", &fakeMySQLError{Number: 1452, Message: "Cannot add or update"}, false},
		{"plain text dup key (Postgres)", errors.New("pq: duplicate key value violates unique constraint \"x\""), true},
		{"plain text mysql", errors.New("Error 1062: Duplicate entry 'y' for key 'PRIMARY'"), true},
		{"plain text sqlite", errors.New("constraint failed: UNIQUE constraint failed: users.email"), true},
		{"unrelated", errors.New("network is unreachable"), false},
		{"nil", nil, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUniqueViolation(tt.err); got != tt.want {
				t.Errorf("IsUniqueViolation(%q) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsUniqueViolation_WrappedError(t *testing.T) {
	// Wrapping must still match — errors.As / errors.Unwrap traversal.
	inner := &fakePgError{code: "23505"}
	wrapped := fmt.Errorf("db op: %w", inner)
	if !IsUniqueViolation(wrapped) {
		t.Error("wrapped pg error should still match")
	}

	wrappedMy := fmt.Errorf("insert: %w", &fakeMySQLError{Number: 1062})
	if !IsUniqueViolation(wrappedMy) {
		t.Error("wrapped mysql error should still match")
	}
}

func TestSlowQueryLogging(t *testing.T) {
	// Just exercise the code path; we don't capture slog output here.
	d := openTestDB(t)
	d.opts.SlowThreshold = time.Nanosecond
	_, _ = d.ExecContext(context.Background(), `SELECT 1`)
}
