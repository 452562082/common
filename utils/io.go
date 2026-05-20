// Package utils provides small, dependency-free helpers shared by the rest of
// the module: a sync.Pool-backed *bytes.Buffer and zero-copy string ⇄ []byte
// converters.
//
// The zero-copy converters use unsafe and are only safe under the rules
// documented on each function. Prefer the standard conversions unless you've
// measured an allocation problem.
package utils

import (
	"bytes"
	"io"
	"sync"
	"unsafe"
)

var bufferPool sync.Pool

// AcquireByteBuffer returns a *bytes.Buffer from the pool.
func AcquireByteBuffer() *bytes.Buffer {
	if v := bufferPool.Get(); v != nil {
		return v.(*bytes.Buffer)
	}
	return bytes.NewBuffer(make([]byte, 0, bytes.MinRead))
}

// ReleaseByteBuffer resets buf and returns it to the pool.
func ReleaseByteBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

// ReadAllToByteBuffer reads from r until EOF, writing into buf, and returns
// buf's bytes. Unlike io.ReadAll, the destination buffer is supplied by the
// caller (typically from AcquireByteBuffer).
func ReadAllToByteBuffer(r io.Reader, buf *bytes.Buffer) (b []byte, err error) {
	defer func() {
		if e := recover(); e != nil {
			if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
				err = panicErr
			} else {
				panic(e)
			}
		}
	}()
	_, err = buf.ReadFrom(r)
	return buf.Bytes(), err
}

// StringToBytes converts a string to a []byte without allocating.
//
// The returned slice MUST NOT be modified — strings are immutable, and
// mutating the backing array is undefined behaviour.
func StringToBytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

// BytesToString converts a []byte to a string without allocating.
//
// The source slice MUST NOT be modified after the conversion — the resulting
// string shares its backing array.
func BytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

// S2B is a short alias for StringToBytes. Deprecated: use StringToBytes.
func S2B(s string) []byte { return StringToBytes(s) }

// B2S is a short alias for BytesToString. Deprecated: use BytesToString.
func B2S(b []byte) string { return BytesToString(b) }
