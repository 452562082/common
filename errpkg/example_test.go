package errpkg_test

import (
	"errors"
	"fmt"
	"io"

	"common/errpkg"
)

// ExampleNew creates a typed error and inspects its Code / Status.
func ExampleNew() {
	err := errpkg.New("USER_NOT_FOUND", 404, "user %s does not exist", "u-1")
	fmt.Println(err)
	fmt.Println("code:", errpkg.CodeOf(err))
	fmt.Println("status:", errpkg.StatusOf(err))
	// Output:
	// user u-1 does not exist
	// code: USER_NOT_FOUND
	// status: 404
}

// ExampleWrap wraps a lower-level error and verifies errors.Is descends through it.
func ExampleWrap() {
	err := errpkg.Wrap(io.EOF, "READ_FAIL", 500, "cannot read user profile")
	fmt.Println("matches io.EOF:", errors.Is(err, io.EOF))
	fmt.Println("code:", errpkg.CodeOf(err))
	// Output:
	// matches io.EOF: true
	// code: READ_FAIL
}

// ExampleNewMulti collects multiple errors and joins them once.
func ExampleNewMulti() {
	m := errpkg.NewMulti()
	m.Append(errors.New("step 1 failed"))
	m.Append(nil) // silently dropped
	m.Append(errors.New("step 2 failed"))

	if err := m.Err(); err != nil {
		fmt.Println(err)
	}
	// Output:
	// step 1 failed
	// step 2 failed
}
