// Package jwt is a small wrapper around golang-jwt/jwt/v5.
//
// It centralises:
//
//   - Algorithm selection (HS256 today, RS256 when NewRSA is used).
//   - A NewClaims helper that pre-fills iat / nbf / exp / iss using the
//     Signer's defaults — composable with any custom claims struct.
//   - A typed Verify that distinguishes ErrExpired from ErrInvalid via
//     errors.Is, so handlers can respond 401 vs 400 cleanly.
//
// Usage:
//
//	type MyClaims struct {
//	    jwt.RegisteredClaims
//	    UserID string   `json:"uid"`
//	    Roles  []string `json:"roles"`
//	}
//
//	s := jwtx.NewHMAC("billing-issuer", []byte("super-secret"), time.Hour)
//	tok, err := jwtx.Sign(s, MyClaims{
//	    RegisteredClaims: s.NewClaims(),
//	    UserID:           "u-1",
//	    Roles:            []string{"admin"},
//	})
//	parsed, err := jwtx.Verify[MyClaims](s, tok)
package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// ErrExpired signals an exp claim in the past.
var ErrExpired = errors.New("jwt: token expired")

// ErrInvalid is returned for any other validation failure (bad signature,
// malformed token, future-dated nbf, ...).
var ErrInvalid = errors.New("jwt: token invalid")

// Algorithm picks the signing scheme.
type Algorithm string

const (
	HS256 Algorithm = "HS256"
	RS256 Algorithm = "RS256"
)

// Signer holds the keys + defaults for token issue and verification.
type Signer struct {
	alg        Algorithm
	signKey    any
	verifyKey  any
	issuer     string
	defaultTTL time.Duration
}

// NewHMAC builds a Signer using HS256 (HMAC-SHA256).
// defaultTTL is the expiry applied to claims built with NewClaims().
func NewHMAC(issuer string, secret []byte, defaultTTL time.Duration) *Signer {
	return &Signer{
		alg:        HS256,
		signKey:    secret,
		verifyKey:  secret,
		issuer:     issuer,
		defaultTTL: defaultTTL,
	}
}

// NewRSA builds a Signer using RS256. priv signs, pub verifies.
func NewRSA(issuer string, priv *rsa.PrivateKey, pub *rsa.PublicKey, defaultTTL time.Duration) *Signer {
	return &Signer{
		alg:        RS256,
		signKey:    priv,
		verifyKey:  pub,
		issuer:     issuer,
		defaultTTL: defaultTTL,
	}
}

// RegisteredClaims is re-exported so callers can embed it without importing
// the underlying library directly.
type RegisteredClaims = gojwt.RegisteredClaims

// Claims is the minimal interface a custom claims struct must satisfy.
type Claims interface {
	gojwt.Claims
}

// NewClaims returns a RegisteredClaims pre-filled with iat / nbf / exp / iss
// based on the Signer's configuration. Embed the result inside your custom
// claims type before calling Sign.
func (s *Signer) NewClaims() RegisteredClaims {
	now := time.Now()
	c := RegisteredClaims{
		Issuer:    s.issuer,
		IssuedAt:  gojwt.NewNumericDate(now),
		NotBefore: gojwt.NewNumericDate(now),
	}
	if s.defaultTTL > 0 {
		c.ExpiresAt = gojwt.NewNumericDate(now.Add(s.defaultTTL))
	}
	return c
}

// NewClaimsWithTTL is like NewClaims but lets the caller override the TTL.
// ttl == 0 leaves ExpiresAt unset (token never expires). Negative ttl produces
// an already-expired token, useful in tests.
func (s *Signer) NewClaimsWithTTL(ttl time.Duration) RegisteredClaims {
	now := time.Now()
	c := RegisteredClaims{
		Issuer:    s.issuer,
		IssuedAt:  gojwt.NewNumericDate(now),
		NotBefore: gojwt.NewNumericDate(now),
	}
	if ttl != 0 {
		c.ExpiresAt = gojwt.NewNumericDate(now.Add(ttl))
	}
	return c
}

// Sign serialises claims into a compact JWT.
func Sign[C Claims](s *Signer, claims C) (string, error) {
	tok := gojwt.NewWithClaims(s.method(), claims)
	signed, err := tok.SignedString(s.signKey)
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, nil
}

// Verify parses and validates raw, returning the decoded claims of type C.
// Errors wrap ErrExpired or ErrInvalid for clean errors.Is checks.
func Verify[C Claims](s *Signer, raw string) (C, error) {
	var zero C
	out := new(C)
	parsed, err := gojwt.ParseWithClaims(raw, anyAsClaims(out), func(t *gojwt.Token) (any, error) {
		if t.Method.Alg() != string(s.alg) {
			return nil, fmt.Errorf("unexpected signing method %s", t.Method.Alg())
		}
		return s.verifyKey, nil
	})
	if err != nil {
		if errors.Is(err, gojwt.ErrTokenExpired) {
			return zero, fmt.Errorf("%w: %v", ErrExpired, err)
		}
		return zero, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	if !parsed.Valid {
		return zero, ErrInvalid
	}
	return *out, nil
}

func (s *Signer) method() gojwt.SigningMethod {
	switch s.alg {
	case RS256:
		return gojwt.SigningMethodRS256
	default:
		return gojwt.SigningMethodHS256
	}
}

// anyAsClaims is a small helper that exists because gojwt requires its
// pointer-to-claims argument to satisfy the gojwt.Claims interface — the
// generic type parameter C already enforces this, but the conversion needs
// an indirection.
func anyAsClaims[C Claims](p *C) gojwt.Claims {
	return any(p).(gojwt.Claims)
}
