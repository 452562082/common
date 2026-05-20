package mail

import (
	"io"
	"mime/quotedprintable"
)

// newQPWriter returns a quoted-printable encoder wrapping w. This is a thin
// alias so callers in this package can use the stdlib without an extra import.
func newQPWriter(w io.Writer) io.WriteCloser {
	return quotedprintable.NewWriter(w)
}
