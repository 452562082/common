package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"strings"
	"testing"
	"time"
)

type myClaims struct {
	RegisteredClaims
	UID  string   `json:"uid"`
	Tags []string `json:"tags"`
}

func newSigner() *Signer {
	return NewHMAC("issuer-test", []byte("very-secret-key"), time.Hour)
}

func TestSignAndVerify_HMAC(t *testing.T) {
	s := newSigner()
	tok, err := Sign(s, myClaims{
		RegisteredClaims: s.NewClaims(),
		UID:              "u-1",
		Tags:             []string{"a", "b"},
	})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if tok == "" || strings.Count(tok, ".") != 2 {
		t.Errorf("malformed token: %s", tok)
	}

	out, err := Verify[myClaims](s, tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if out.UID != "u-1" || len(out.Tags) != 2 || out.Tags[0] != "a" {
		t.Errorf("decoded wrong: %+v", out)
	}
	if out.Issuer != "issuer-test" {
		t.Errorf("Issuer = %q", out.Issuer)
	}
}

func TestVerify_Expired(t *testing.T) {
	s := newSigner()
	tok, err := Sign(s, myClaims{
		// Issue a token that has already expired.
		RegisteredClaims: s.NewClaimsWithTTL(-time.Minute),
		UID:              "u-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = Verify[myClaims](s, tok)
	if !errors.Is(err, ErrExpired) {
		t.Errorf("expected ErrExpired, got %v", err)
	}
}

func TestVerify_BadSignature(t *testing.T) {
	s1 := NewHMAC("iss", []byte("key-one"), time.Hour)
	s2 := NewHMAC("iss", []byte("key-two"), time.Hour)

	tok, _ := Sign(s1, myClaims{RegisteredClaims: s1.NewClaims(), UID: "u"})
	_, err := Verify[myClaims](s2, tok)
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("expected ErrInvalid, got %v", err)
	}
}

func TestVerify_Malformed(t *testing.T) {
	s := newSigner()
	_, err := Verify[myClaims](s, "not-a-jwt")
	if !errors.Is(err, ErrInvalid) {
		t.Errorf("expected ErrInvalid, got %v", err)
	}
}

func TestVerify_WrongAlgorithm(t *testing.T) {
	hmacSigner := newSigner()
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	rsaSigner := NewRSA("iss", priv, &priv.PublicKey, time.Hour)

	tok, _ := Sign(rsaSigner, myClaims{RegisteredClaims: rsaSigner.NewClaims(), UID: "u"})
	if _, err := Verify[myClaims](hmacSigner, tok); !errors.Is(err, ErrInvalid) {
		t.Errorf("HMAC verifier should reject RS256 token: %v", err)
	}
}

func TestRSA_RoundTrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	s := NewRSA("rsa-iss", priv, &priv.PublicKey, time.Hour)

	tok, err := Sign(s, myClaims{RegisteredClaims: s.NewClaims(), UID: "u-rsa"})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	out, err := Verify[myClaims](s, tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if out.UID != "u-rsa" {
		t.Errorf("UID = %q", out.UID)
	}
}

func TestNewClaims_Defaults(t *testing.T) {
	s := newSigner()
	c := s.NewClaims()
	if c.Issuer != "issuer-test" {
		t.Errorf("Issuer = %q", c.Issuer)
	}
	if c.IssuedAt == nil || c.NotBefore == nil || c.ExpiresAt == nil {
		t.Errorf("expected iat/nbf/exp set, got %+v", c)
	}
	if got := c.ExpiresAt.Sub(c.IssuedAt.Time); got != time.Hour {
		t.Errorf("default TTL not applied: %v", got)
	}
}

func TestNewClaimsWithTTL(t *testing.T) {
	s := newSigner()
	c := s.NewClaimsWithTTL(15 * time.Minute)
	if got := c.ExpiresAt.Sub(c.IssuedAt.Time); got != 15*time.Minute {
		t.Errorf("custom TTL not applied: %v", got)
	}
}
