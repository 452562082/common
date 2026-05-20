// Package errpkg layers a small typed-error model on top of the stdlib errors
// package:
//
//   - Error carries an application Code, an HTTP/gRPC status hint, a
//     human-readable message, an optional cause (errors.Unwrap target), and a
//     captured stack frame from the construction site.
//   - Errors built with New / Wrap satisfy errors.Is and errors.As.
//   - Multi joins independent errors (think Promise.allSettled), and is
//     compatible with errors.Is / errors.As across all wrapped errors via
//     stdlib's errors.Join semantics.
//
// Quick reference:
//
//	var ErrNotFound = errpkg.NewSentinel("USER_NOT_FOUND", 404, "user not found")
//
//	if u == nil {
//	    return ErrNotFound
//	}
//	err := errpkg.Wrap(io.EOF, "READ_FAIL", 500, "cannot read user")
//	if errors.Is(err, io.EOF) { ... }
package errpkg

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
)

// Code is an opaque application error code, conventionally upper-snake-case.
// It is the stable identifier clients use to dispatch on error types — keep
// codes namespaced and don't recycle them.
type Code string

// Error is the typed error produced by New / Wrap.
type Error struct {
	code    Code
	status  int    // HTTP status when the error reaches an HTTP handler
	message string // human-readable
	cause   error  // errors.Unwrap target
	stack   []uintptr
}

// New creates an Error with no cause.
func New(code Code, status int, format string, args ...any) *Error {
	return build(nil, code, status, fmt.Sprintf(format, args...))
}

// Wrap creates an Error that wraps cause. errors.Is/As traverse through cause.
// Passing nil cause makes Wrap equivalent to New.
func Wrap(cause error, code Code, status int, format string, args ...any) *Error {
	return build(cause, code, status, fmt.Sprintf(format, args...))
}

// NewSentinel is a convenience for declaring package-level sentinel errors.
// The returned value has no captured stack (because there's no useful frame
// to capture at init time) but is otherwise identical.
func NewSentinel(code Code, status int, message string) *Error {
	return &Error{code: code, status: status, message: message}
}

// stackDepth controls the maximum number of frames captured by build.
// Atomic to permit lock-free reads (called on every New / Wrap).
var stackDepth atomic.Int32

func init() { stackDepth.Store(32) }

// SetStackDepth changes the maximum number of frames captured on subsequent
// New / Wrap calls. Pass 0 to disable stack capture entirely (useful in
// hot paths that wrap and immediately log).
func SetStackDepth(n int) {
	if n < 0 {
		n = 0
	}
	stackDepth.Store(int32(n))
}

// StackDepth returns the current limit.
func StackDepth() int { return int(stackDepth.Load()) }

func build(cause error, code Code, status int, message string) *Error {
	e := &Error{
		code:    code,
		status:  status,
		message: message,
		cause:   cause,
	}
	depth := int(stackDepth.Load())
	if depth > 0 {
		pcs := make([]uintptr, depth)
		n := runtime.Callers(3, pcs)
		e.stack = pcs[:n]
	}
	return e
}

// Error implements the error interface. It includes the cause for log/debug
// readability; programmatic callers should use Code / Status / Unwrap.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.cause == nil {
		return e.message
	}
	return e.message + ": " + e.cause.Error()
}

// Unwrap returns the wrapped cause, satisfying errors.Unwrap.
func (e *Error) Unwrap() error { return e.cause }

// Code returns the application error code.
func (e *Error) Code() Code { return e.code }

// Status returns the HTTP-style status (200..599) hint. May be 0 if not set.
func (e *Error) Status() int { return e.status }

// Message returns the human-readable description without the cause appended.
func (e *Error) Message() string { return e.message }

// Stack returns the captured frames as a readable, multi-line string.
// Returns "" when no stack was captured (e.g. sentinels).
func (e *Error) Stack() string {
	if len(e.stack) == 0 {
		return ""
	}
	frames := runtime.CallersFrames(e.stack)
	var sb strings.Builder
	for {
		fr, more := frames.Next()
		if fr.Function == "" {
			break
		}
		fmt.Fprintf(&sb, "%s\n\t%s:%d\n", fr.Function, fr.File, fr.Line)
		if !more {
			break
		}
	}
	return sb.String()
}

// Is supports errors.Is matching:
//   - Two *Error values match iff their codes are equal.
//   - errors.Is also descends into cause via Unwrap.
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return e.code != "" && e.code == t.code
	}
	return false
}

// CodeOf returns the Code of the first *Error in err's chain, or "".
func CodeOf(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.code
	}
	return ""
}

// StatusOf returns the Status hint of the first *Error in err's chain.
// Returns 0 when no *Error is found; HTTP handlers typically map 0 → 500.
func StatusOf(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.status
	}
	return 0
}

// ---------- Multi-error ------------------------------------------------------

// Multi accumulates multiple errors and presents them as a single error.
// It is safe to use with the stdlib errors.Is / errors.As, since it
// delegates to errors.Join semantics under the hood.
//
// Use Multi when running a batch of operations in parallel and you want to
// surface every failure, not just the first.
type Multi struct {
	errs []error
}

// NewMulti returns an empty Multi.
func NewMulti() *Multi { return &Multi{} }

// Append adds non-nil errors to the collector.
func (m *Multi) Append(errs ...error) {
	for _, e := range errs {
		if e != nil {
			m.errs = append(m.errs, e)
		}
	}
}

// Len reports the number of accumulated errors.
func (m *Multi) Len() int { return len(m.errs) }

// Err returns nil when no errors were appended, otherwise an error that
// implements `interface{ Unwrap() []error }` (compatible with errors.Is/As).
func (m *Multi) Err() error {
	if len(m.errs) == 0 {
		return nil
	}
	if len(m.errs) == 1 {
		return m.errs[0]
	}
	return errors.Join(m.errs...)
}

// Errors returns the accumulated slice (read-only by convention).
func (m *Multi) Errors() []error { return m.errs }

// Append is a free-function form for the common idiom:
//
//	var errs error
//	errs = errpkg.AppendErr(errs, doStep1())
//	errs = errpkg.AppendErr(errs, doStep2())
//	return errs
func AppendErr(dst, err error) error {
	if err == nil {
		return dst
	}
	if dst == nil {
		return err
	}
	return errors.Join(dst, err)
}
