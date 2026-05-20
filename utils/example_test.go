package utils_test

import (
	"bytes"
	"fmt"

	"common/utils"
)

// ExampleStringToBytes shows the zero-copy conversion.
//
// Note: the returned slice MUST NOT be modified — Go strings are immutable
// and the slice shares their backing array.
func ExampleStringToBytes() {
	b := utils.StringToBytes("hello")
	fmt.Println(len(b), string(b))
	// Output:
	// 5 hello
}

// ExampleAcquireByteBuffer demonstrates the pooled buffer pattern.
func ExampleAcquireByteBuffer() {
	buf := utils.AcquireByteBuffer()
	defer utils.ReleaseByteBuffer(buf)

	buf.WriteString("payload-1")
	fmt.Println(bytes.Equal(buf.Bytes(), []byte("payload-1")))
	// Output:
	// true
}
