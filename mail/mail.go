// Package mail is a small SMTP client built on stdlib net/smtp + a hand-rolled
// MIME composer.
//
// Why not import gomail / go-mail? Because the protocol-level needs of a
// typical service (TLS, AUTH PLAIN, text + HTML alternative, attachments)
// can be written in ~200 LOC and avoid pulling in a third-party dep just to
// send a few transactional emails.
//
// Usage:
//
//	s := mail.NewSender(mail.Options{
//	    Host:     "smtp.example.com",
//	    Port:     587,
//	    Username: "alerts@example.com",
//	    Password: os.Getenv("SMTP_PASSWORD"),
//	    From:     "alerts@example.com",
//	    UseTLS:   true,
//	})
//
//	err := s.Send(ctx, mail.Message{
//	    To:      []string{"oncall@example.com"},
//	    Subject: "DB latency spike",
//	    Text:    "p99 > 500ms for 5 min, see runbook X.",
//	    HTML:    "<p>p99 > 500ms for 5 min — <a href='...'>runbook</a></p>",
//	    Attachments: []mail.Attachment{
//	        {Filename: "graph.png", Content: pngBytes, ContentType: "image/png"},
//	    },
//	})
package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net/smtp"
	"strings"
	"time"
)

// Options configures NewSender.
type Options struct {
	Host string // SMTP server host
	Port int    // SMTP server port (25, 465, 587, ...)

	Username string
	Password string

	// From is the default From header. May be overridden per message.
	From string

	// UseTLS picks STARTTLS upgrade after EHLO. Use this for port 587.
	UseTLS bool

	// UseTLSImplicit dials a TLS connection from the start (SMTPS, port 465).
	UseTLSImplicit bool

	// InsecureSkipVerify disables TLS certificate verification.
	// Only set during development / self-signed setups.
	InsecureSkipVerify bool

	// Timeout caps the whole Send. Default 30s.
	Timeout time.Duration
}

// Sender wraps a connection's worth of state. It is safe for concurrent use:
// each Send opens its own connection.
type Sender struct {
	opts Options
	auth smtp.Auth
}

// NewSender validates opts and returns a ready-to-use Sender.
func NewSender(opts Options) (*Sender, error) {
	if opts.Host == "" {
		return nil, errors.New("mail: Host is required")
	}
	if opts.Port == 0 {
		return nil, errors.New("mail: Port is required")
	}
	if opts.From == "" {
		return nil, errors.New("mail: From is required")
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	var auth smtp.Auth
	if opts.Username != "" || opts.Password != "" {
		// PLAIN auth ships the password as base64-encoded plaintext.
		// Refuse to do that over an unencrypted connection — silently leaking
		// credentials is a worse failure mode than refusing to start.
		if !opts.UseTLS && !opts.UseTLSImplicit {
			return nil, errors.New("mail: credentials require UseTLS or UseTLSImplicit")
		}
		auth = smtp.PlainAuth("", opts.Username, opts.Password, opts.Host)
	}
	return &Sender{opts: opts, auth: auth}, nil
}

// containsCRLF reports whether s contains a CR or LF byte. We use this to
// reject header injection attempts before we serialise the MIME message.
func containsCRLF(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' || s[i] == '\n' {
			return true
		}
	}
	return false
}

// Attachment is a single file attached to a Message.
type Attachment struct {
	Filename    string
	Content     []byte
	ContentType string // defaults to application/octet-stream
}

// Message describes one outgoing email.
type Message struct {
	From    string // overrides Sender.From if set
	To      []string
	CC      []string
	BCC     []string
	Subject string
	Text    string // plain-text alternative (always recommended)
	HTML    string // optional HTML alternative
	Headers map[string]string

	Attachments []Attachment
}

// Send dispatches msg. ctx caps the operation; on cancellation the connection
// is torn down.
func (s *Sender) Send(ctx context.Context, msg Message) error {
	if err := msg.validate(); err != nil {
		return err
	}
	from := msg.From
	if from == "" {
		from = s.opts.From
	}
	if containsCRLF(from) {
		return errors.New("mail: From contains CR/LF")
	}

	body, err := buildMIME(from, msg)
	if err != nil {
		return fmt.Errorf("mail: build mime: %w", err)
	}

	deadline, cancel := context.WithTimeout(ctx, s.opts.Timeout)
	defer cancel()

	type result struct{ err error }
	done := make(chan result, 1)
	go func() {
		done <- result{err: s.send(from, msg.recipients(), body)}
	}()
	select {
	case <-deadline.Done():
		return fmt.Errorf("mail: %w", deadline.Err())
	case r := <-done:
		if r.err != nil {
			return fmt.Errorf("mail: %w", r.err)
		}
		return nil
	}
}

func (m Message) validate() error {
	if len(m.To) == 0 && len(m.CC) == 0 && len(m.BCC) == 0 {
		return errors.New("mail: no recipients")
	}
	if m.Subject == "" {
		return errors.New("mail: Subject is required")
	}
	if m.Text == "" && m.HTML == "" {
		return errors.New("mail: Text or HTML body is required")
	}
	// CRLF injection guard: any field that ends up in a header MUST NOT
	// contain CR / LF. Subject is run through QEncoding so it's safe, but
	// addresses and arbitrary custom headers are not encoded.
	if containsCRLF(m.From) {
		return errors.New("mail: From contains CR/LF")
	}
	for _, addrList := range [][]string{m.To, m.CC, m.BCC} {
		for _, a := range addrList {
			if containsCRLF(a) {
				return fmt.Errorf("mail: recipient %q contains CR/LF", a)
			}
		}
	}
	for k, v := range m.Headers {
		if containsCRLF(k) || containsCRLF(v) {
			return fmt.Errorf("mail: header %q contains CR/LF", k)
		}
	}
	for _, a := range m.Attachments {
		if containsCRLF(a.Filename) {
			return fmt.Errorf("mail: attachment filename %q contains CR/LF", a.Filename)
		}
	}
	return nil
}

func (m Message) recipients() []string {
	all := make([]string, 0, len(m.To)+len(m.CC)+len(m.BCC))
	all = append(all, m.To...)
	all = append(all, m.CC...)
	all = append(all, m.BCC...)
	return all
}

func (s *Sender) send(from string, to []string, body []byte) error {
	addr := fmt.Sprintf("%s:%d", s.opts.Host, s.opts.Port)
	tlsConfig := &tls.Config{
		ServerName:         s.opts.Host,
		InsecureSkipVerify: s.opts.InsecureSkipVerify, //nolint:gosec
	}

	var client *smtp.Client
	var err error
	if s.opts.UseTLSImplicit {
		conn, dialErr := tls.Dial("tcp", addr, tlsConfig)
		if dialErr != nil {
			return fmt.Errorf("tls dial: %w", dialErr)
		}
		client, err = smtp.NewClient(conn, s.opts.Host)
	} else {
		client, err = smtp.Dial(addr)
	}
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer client.Close()

	if err := client.Hello(localName(s.opts.From)); err != nil {
		return fmt.Errorf("hello: %w", err)
	}

	if s.opts.UseTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsConfig); err != nil {
				return fmt.Errorf("starttls: %w", err)
			}
		}
	}

	if s.auth != nil {
		if err := client.Auth(s.auth); err != nil {
			return fmt.Errorf("auth: %w", err)
		}
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("mail-from: %w", err)
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("rcpt %s: %w", rcpt, err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(body); err != nil {
		_ = w.Close()
		return fmt.Errorf("write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close body: %w", err)
	}
	return client.Quit()
}

// localName extracts the part after "@" of an email for use as the EHLO hostname.
// Falls back to "localhost" when no @ is present.
func localName(addr string) string {
	if i := strings.IndexByte(addr, '@'); i >= 0 && i+1 < len(addr) {
		return addr[i+1:]
	}
	return "localhost"
}

// buildMIME constructs a multipart/mixed message with an alternative
// (text/plain + text/html) main body and optional attachments.
func buildMIME(from string, m Message) ([]byte, error) {
	var buf bytes.Buffer

	writeHeader := func(k, v string) { fmt.Fprintf(&buf, "%s: %s\r\n", k, v) }

	writeHeader("From", from)
	writeHeader("To", strings.Join(m.To, ", "))
	if len(m.CC) > 0 {
		writeHeader("Cc", strings.Join(m.CC, ", "))
	}
	writeHeader("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	writeHeader("MIME-Version", "1.0")
	writeHeader("Date", time.Now().Format(time.RFC1123Z))
	for k, v := range m.Headers {
		writeHeader(k, v)
	}

	mixedBoundary := randomBoundary("mixed")
	altBoundary := randomBoundary("alt")

	if len(m.Attachments) > 0 {
		writeHeader("Content-Type", `multipart/mixed; boundary="`+mixedBoundary+`"`)
		buf.WriteString("\r\n")
		writePart(&buf, mixedBoundary, false)
		writeAltBlock(&buf, altBoundary, m)
		for _, a := range m.Attachments {
			writePart(&buf, mixedBoundary, false)
			writeAttachment(&buf, a)
		}
		writePart(&buf, mixedBoundary, true)
	} else {
		writeAltBlockSelfContained(&buf, altBoundary, m)
	}
	return buf.Bytes(), nil
}

func writeAltBlock(buf *bytes.Buffer, boundary string, m Message) {
	fmt.Fprintf(buf, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary)
	if m.Text != "" {
		writePart(buf, boundary, false)
		buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		buf.WriteString(qpEncode(m.Text))
		buf.WriteString("\r\n")
	}
	if m.HTML != "" {
		writePart(buf, boundary, false)
		buf.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		buf.WriteString(qpEncode(m.HTML))
		buf.WriteString("\r\n")
	}
	writePart(buf, boundary, true)
}

func writeAltBlockSelfContained(buf *bytes.Buffer, boundary string, m Message) {
	if m.HTML == "" {
		// Plain text only — no multipart wrapper needed.
		buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		buf.WriteString(qpEncode(m.Text))
		return
	}
	fmt.Fprintf(buf, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", boundary)
	if m.Text != "" {
		writePart(buf, boundary, false)
		buf.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		buf.WriteString(qpEncode(m.Text))
		buf.WriteString("\r\n")
	}
	writePart(buf, boundary, false)
	buf.WriteString("Content-Type: text/html; charset=\"utf-8\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
	buf.WriteString(qpEncode(m.HTML))
	buf.WriteString("\r\n")
	writePart(buf, boundary, true)
}

func writeAttachment(buf *bytes.Buffer, a Attachment) {
	ct := a.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	fmt.Fprintf(buf, "Content-Type: %s\r\n", ct)
	fmt.Fprintf(buf, "Content-Disposition: attachment; filename=\"%s\"\r\n",
		mime.QEncoding.Encode("utf-8", a.Filename))
	buf.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	encoded := base64.StdEncoding.EncodeToString(a.Content)
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		buf.WriteString(encoded[i:end])
		buf.WriteString("\r\n")
	}
}

func writePart(buf *bytes.Buffer, boundary string, last bool) {
	if last {
		fmt.Fprintf(buf, "--%s--\r\n", boundary)
		return
	}
	fmt.Fprintf(buf, "--%s\r\n", boundary)
}

// qpEncode runs a tiny quoted-printable encoder. The stdlib has
// mime/quotedprintable, which we re-use here.
func qpEncode(s string) string {
	var out bytes.Buffer
	w := newQPWriter(&out)
	_, _ = w.Write([]byte(s))
	_ = w.Close()
	return out.String()
}

var (
	_ = strings.Builder{} // keep imports tidy if compiler trims
)

// randomBoundary produces a stable-ish MIME boundary token. We use a hash of
// time.Now() ns + a tag so different parts in the same message don't collide.
func randomBoundary(tag string) string {
	now := time.Now().UnixNano()
	return fmt.Sprintf("=_%s_%x", tag, now)
}
