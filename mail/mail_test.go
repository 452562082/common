package mail

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewSender_Validation(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{"missing host", Options{Port: 25, From: "a@b"}},
		{"missing port", Options{Host: "h", From: "a@b"}},
		{"missing from", Options{Host: "h", Port: 25}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewSender(tt.opts); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestNewSender_DefaultsTimeout(t *testing.T) {
	s, err := NewSender(Options{Host: "h", Port: 25, From: "a@b"})
	if err != nil {
		t.Fatal(err)
	}
	if s.opts.Timeout == 0 {
		t.Error("expected non-zero default timeout")
	}
}

func TestNewSender_CredentialsRequireTLS(t *testing.T) {
	// PLAIN auth over a non-TLS connection must be rejected.
	_, err := NewSender(Options{
		Host: "h", Port: 25, From: "a@b",
		Username: "u", Password: "p",
	})
	if err == nil {
		t.Error("expected error: credentials without TLS")
	}

	// With UseTLS it must succeed.
	_, err = NewSender(Options{
		Host: "h", Port: 587, From: "a@b",
		Username: "u", Password: "p",
		UseTLS: true,
	})
	if err != nil {
		t.Errorf("UseTLS=true should be accepted: %v", err)
	}
}

func TestMessage_RejectsCRLFInjection(t *testing.T) {
	tests := []struct {
		name string
		msg  Message
	}{
		{"To injection", Message{
			To:      []string{"victim@x\r\nBcc: attacker@evil"},
			Subject: "S", Text: "t",
		}},
		{"CC injection", Message{
			To: []string{"v@x"}, CC: []string{"c\r\nX-Spoof: yes"},
			Subject: "S", Text: "t",
		}},
		{"BCC injection", Message{
			To: []string{"v@x"}, BCC: []string{"c\nReply-To: phish@x"},
			Subject: "S", Text: "t",
		}},
		{"From injection", Message{
			From: "a@b\r\nBcc: leak@evil",
			To:   []string{"v@x"}, Subject: "S", Text: "t",
		}},
		{"Custom header injection", Message{
			To: []string{"v@x"}, Subject: "S", Text: "t",
			Headers: map[string]string{"X-Marker": "ok\r\nBcc: leak@x"},
		}},
		{"Attachment filename injection", Message{
			To: []string{"v@x"}, Subject: "S", Text: "t",
			Attachments: []Attachment{{Filename: "ok\r\nX-Spoof: yes", Content: []byte("x")}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.msg.validate(); err == nil {
				t.Errorf("expected validate() to reject CRLF: %+v", tt.msg)
			}
		})
	}
}

func TestMessage_Validate(t *testing.T) {
	cases := []struct {
		name string
		msg  Message
	}{
		{"no recipients", Message{Subject: "x", Text: "y"}},
		{"no subject", Message{To: []string{"a@b"}, Text: "y"}},
		{"no body", Message{To: []string{"a@b"}, Subject: "s"}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.msg.validate(); err == nil {
				t.Error("expected error")
			}
		})
	}

	good := Message{To: []string{"a@b"}, Subject: "s", Text: "body"}
	if err := good.validate(); err != nil {
		t.Errorf("good message should validate: %v", err)
	}
}

func TestBuildMIME_PlainText(t *testing.T) {
	body, err := buildMIME("alice@example.com", Message{
		To:      []string{"bob@example.com"},
		Subject: "Hello",
		Text:    "Plain content with = and é",
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "From: alice@example.com") {
		t.Errorf("From header missing")
	}
	if !strings.Contains(s, "To: bob@example.com") {
		t.Errorf("To header missing")
	}
	if !strings.Contains(s, "Content-Type: text/plain") {
		t.Errorf("plain content-type missing:\n%s", s)
	}
	// "=" must be quoted-printable encoded
	if !strings.Contains(s, "=3D") {
		t.Errorf("QP encoding missing for '='; body:\n%s", s)
	}
}

func TestBuildMIME_HTMLAlternative(t *testing.T) {
	body, err := buildMIME("alice@example.com", Message{
		To:      []string{"bob@example.com"},
		Subject: "Hi",
		Text:    "text",
		HTML:    "<p>hi</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "multipart/alternative") {
		t.Errorf("expected multipart/alternative for HTML+text:\n%s", s)
	}
	if !strings.Contains(s, "text/plain") || !strings.Contains(s, "text/html") {
		t.Errorf("both content types should appear")
	}
}

func TestBuildMIME_Attachments(t *testing.T) {
	body, err := buildMIME("alice@example.com", Message{
		To:      []string{"bob@example.com"},
		Subject: "Hi",
		Text:    "see attached",
		Attachments: []Attachment{
			{Filename: "hello.txt", Content: []byte("hello world"), ContentType: "text/plain"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "multipart/mixed") {
		t.Errorf("expected multipart/mixed with attachments")
	}
	if !strings.Contains(s, "Content-Disposition: attachment") {
		t.Errorf("attachment disposition missing")
	}
	if !strings.Contains(s, "Content-Transfer-Encoding: base64") {
		t.Errorf("base64 encoding for attachment missing")
	}
}

func TestBuildMIME_CCHeaders(t *testing.T) {
	body, _ := buildMIME("a@b.com", Message{
		To: []string{"x@y"}, CC: []string{"c@d"},
		Subject: "S", Text: "t",
	})
	if !bytes.Contains(body, []byte("Cc: c@d")) {
		t.Errorf("Cc header missing")
	}
}

func TestRecipients(t *testing.T) {
	m := Message{To: []string{"a"}, CC: []string{"b"}, BCC: []string{"c"}}
	got := m.recipients()
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Errorf("recipients = %v", got)
	}
}

func TestLocalName(t *testing.T) {
	if localName("a@example.com") != "example.com" {
		t.Errorf("got %q", localName("a@example.com"))
	}
	if localName("noaddr") != "localhost" {
		t.Errorf("got %q", localName("noaddr"))
	}
}
