package errpkg

import (
	"errors"
	"io"
	"strings"
	"testing"
)

var ErrSentinel = NewSentinel("TEST_SENTINEL", 404, "sentinel error")

func TestNew_Basic(t *testing.T) {
	e := New("TEST_CODE", 400, "bad %s", "input")
	if e.Code() != "TEST_CODE" {
		t.Errorf("Code = %q", e.Code())
	}
	if e.Status() != 400 {
		t.Errorf("Status = %d", e.Status())
	}
	if e.Message() != "bad input" {
		t.Errorf("Message = %q", e.Message())
	}
	if e.Error() != "bad input" {
		t.Errorf("Error = %q", e.Error())
	}
}

func TestWrap_CauseAndUnwrap(t *testing.T) {
	wrapped := Wrap(io.EOF, "READ_FAIL", 500, "cannot read")
	if !errors.Is(wrapped, io.EOF) {
		t.Error("errors.Is should descend into cause")
	}
	if !strings.Contains(wrapped.Error(), "cannot read") || !strings.Contains(wrapped.Error(), "EOF") {
		t.Errorf("Error message lost detail: %q", wrapped.Error())
	}
}

func TestWrap_NilCauseEqualsNew(t *testing.T) {
	e := Wrap(nil, "X", 0, "msg")
	if e.Unwrap() != nil {
		t.Error("nil cause should unwrap to nil")
	}
}

func TestIs_BySentinel(t *testing.T) {
	if !errors.Is(ErrSentinel, ErrSentinel) {
		t.Error("sentinel must match itself")
	}
	// A new error with the same code matches the sentinel.
	e := New("TEST_SENTINEL", 500, "another instance")
	if !errors.Is(e, ErrSentinel) {
		t.Error("same-code instance should match the sentinel")
	}
	// Different code should NOT match.
	if errors.Is(New("OTHER", 0, ""), ErrSentinel) {
		t.Error("different code should not match")
	}
}

func TestIs_AcrossWrap(t *testing.T) {
	wrapped := Wrap(ErrSentinel, "CTX_CODE", 500, "context")
	if !errors.Is(wrapped, ErrSentinel) {
		t.Error("Is must descend through Wrap")
	}
}

func TestCodeOf_StatusOf(t *testing.T) {
	e := New("X", 418, "teapot")
	if CodeOf(e) != "X" {
		t.Errorf("CodeOf = %q", CodeOf(e))
	}
	if StatusOf(e) != 418 {
		t.Errorf("StatusOf = %d", StatusOf(e))
	}
	if CodeOf(io.EOF) != "" {
		t.Error("CodeOf of non-*Error should be empty")
	}
	if StatusOf(io.EOF) != 0 {
		t.Error("StatusOf of non-*Error should be 0")
	}
}

func TestStack(t *testing.T) {
	e := New("X", 0, "msg")
	if !strings.Contains(e.Stack(), "TestStack") {
		t.Errorf("stack should mention TestStack:\n%s", e.Stack())
	}
	if ErrSentinel.Stack() != "" {
		t.Error("sentinels should have empty stacks")
	}
}

func TestMulti_AppendAndErr(t *testing.T) {
	m := NewMulti()
	if m.Err() != nil {
		t.Error("empty multi should return nil")
	}
	m.Append(nil, errors.New("a"), nil, errors.New("b"))
	if m.Len() != 2 {
		t.Errorf("Len = %d", m.Len())
	}
	err := m.Err()
	if err == nil || !strings.Contains(err.Error(), "a") || !strings.Contains(err.Error(), "b") {
		t.Errorf("Err = %v", err)
	}
}

func TestMulti_SingleErrorUnwrapped(t *testing.T) {
	sentinel := errors.New("only one")
	m := NewMulti()
	m.Append(sentinel)
	if m.Err() != sentinel {
		t.Errorf("single-error Multi should return that error unchanged, got %v", m.Err())
	}
}

func TestMulti_IsTraversesJoined(t *testing.T) {
	m := NewMulti()
	m.Append(errors.New("x"), io.EOF, errors.New("y"))
	if !errors.Is(m.Err(), io.EOF) {
		t.Error("errors.Is should match any joined error")
	}
}

func TestSetStackDepth(t *testing.T) {
	orig := StackDepth()
	defer SetStackDepth(orig)

	SetStackDepth(0)
	e := New("X", 0, "msg")
	if e.Stack() != "" {
		t.Errorf("depth=0 should disable stack capture, got:\n%s", e.Stack())
	}

	SetStackDepth(64)
	e = New("X", 0, "msg")
	if e.Stack() == "" {
		t.Error("depth=64 should produce a non-empty stack")
	}

	SetStackDepth(-1) // clamped to 0
	if StackDepth() != 0 {
		t.Errorf("negative depth should clamp to 0, got %d", StackDepth())
	}
}

func TestAppendErr(t *testing.T) {
	e := AppendErr(nil, errors.New("a"))
	if e == nil || e.Error() != "a" {
		t.Errorf("first append: %v", e)
	}
	e = AppendErr(e, errors.New("b"))
	if !strings.Contains(e.Error(), "a") || !strings.Contains(e.Error(), "b") {
		t.Errorf("second append lost data: %v", e)
	}
	if AppendErr(nil, nil) != nil {
		t.Error("AppendErr(nil, nil) should be nil")
	}
}
