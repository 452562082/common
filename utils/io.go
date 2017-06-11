package utils

import (
	"bytes"
	"io"
	"sync"
)

var bufferPool sync.Pool

func AcquireByteBuffer() *bytes.Buffer {
	v := bufferPool.Get()
	if v == nil {
		return bytes.NewBuffer(make([]byte, 0, bytes.MinRead))
	}

	buf := v.(*bytes.Buffer)
	return buf
}

func ReleaseByteBuffer(buf *bytes.Buffer) {
	buf.Reset()
	bufferPool.Put(buf)
}

// ReadAllToByteBuffer reads from r until an error or EOF and returns the data it read.
// A successful call returns err == nil, not err == EOF. Because ReadAllToByteBuffer is
// defined to read from src until EOF, it does not treat an EOF from Read as an error to be reported.
func ReadAllToByteBuffer(r io.Reader, buf *bytes.Buffer) (b []byte, err error) {
	// If the buffer overflows, we will get bytes.ErrTooLarge.
	// Return that as an error. Any other panic remains.
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
			err = panicErr
		} else {
			panic(e)
		}
	}()
	_, err = buf.ReadFrom(r)
	return buf.Bytes(), err
}
