package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"sync"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// KeySet supports key rotation: tokens carry a `kid` header that names the
// key used to sign them, and Verify looks up the matching key in the set.
//
// Two operating modes coexist on one KeySet:
//
//   - Sign: tokens are signed with the *active* key (set by SetActiveKey or
//     the most recently added one). The matching kid is written into the
//     token's header.
//   - Verify: the kid from the token header picks the verification key.
//     Tokens without a kid are rejected.
//
// Adding a new key + flipping the active key + removing the retired key
// covers the standard "rotate keys without downtime" sequence:
//
//	ks := jwtx.NewKeySet("billing-iss", time.Hour)
//	_ = ks.AddHMACKey("k1", []byte("old-secret"))
//	_ = ks.AddHMACKey("k2", []byte("new-secret"))
//	ks.SetActive("k2")                  // new tokens use k2
//	// ... wait until every old k1 token has expired ...
//	ks.Remove("k1")                     // retire k1
type KeySet struct {
	issuer     string
	defaultTTL time.Duration

	mu     sync.RWMutex
	keys   map[string]*keyEntry
	active string
}

type keyEntry struct {
	alg       Algorithm
	signKey   any
	verifyKey any
}

// NewKeySet returns an empty KeySet. Add at least one key + call SetActive
// before signing.
func NewKeySet(issuer string, defaultTTL time.Duration) *KeySet {
	return &KeySet{
		issuer:     issuer,
		defaultTTL: defaultTTL,
		keys:       make(map[string]*keyEntry),
	}
}

// AddHMACKey registers a HS256 key under kid. If no key is currently active,
// the newly-added one becomes active.
func (ks *KeySet) AddHMACKey(kid string, secret []byte) error {
	if kid == "" {
		return errors.New("jwt: kid is required")
	}
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.keys[kid] = &keyEntry{alg: HS256, signKey: secret, verifyKey: secret}
	if ks.active == "" {
		ks.active = kid
	}
	return nil
}

// AddRSAKey registers an RS256 key pair under kid.
func (ks *KeySet) AddRSAKey(kid string, priv *rsa.PrivateKey, pub *rsa.PublicKey) error {
	if kid == "" {
		return errors.New("jwt: kid is required")
	}
	if pub == nil {
		return errors.New("jwt: RSA public key is required")
	}
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.keys[kid] = &keyEntry{alg: RS256, signKey: priv, verifyKey: pub}
	if ks.active == "" {
		ks.active = kid
	}
	return nil
}

// AddVerifyOnlyRSAKey registers a public key under kid that can only verify
// (no signing). Useful for accepting tokens from another service.
func (ks *KeySet) AddVerifyOnlyRSAKey(kid string, pub *rsa.PublicKey) error {
	if kid == "" {
		return errors.New("jwt: kid is required")
	}
	if pub == nil {
		return errors.New("jwt: RSA public key is required")
	}
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.keys[kid] = &keyEntry{alg: RS256, verifyKey: pub}
	return nil
}

// SetActive switches the signing key. Returns an error if kid was never added.
func (ks *KeySet) SetActive(kid string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	if _, ok := ks.keys[kid]; !ok {
		return fmt.Errorf("jwt: kid %q not found", kid)
	}
	ks.active = kid
	return nil
}

// Remove drops kid from the set. After this, tokens signed with that kid
// fail verification with ErrInvalid.
func (ks *KeySet) Remove(kid string) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	delete(ks.keys, kid)
	if ks.active == kid {
		ks.active = ""
	}
}

// ActiveKID returns the kid currently used for signing, or "".
func (ks *KeySet) ActiveKID() string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.active
}

// NewClaims returns a RegisteredClaims with iat / nbf / exp / iss populated.
func (ks *KeySet) NewClaims() RegisteredClaims {
	return ks.NewClaimsWithTTL(ks.defaultTTL)
}

// NewClaimsWithTTL is like NewClaims with an explicit TTL.
func (ks *KeySet) NewClaimsWithTTL(ttl time.Duration) RegisteredClaims {
	now := time.Now()
	c := RegisteredClaims{
		Issuer:    ks.issuer,
		IssuedAt:  gojwt.NewNumericDate(now),
		NotBefore: gojwt.NewNumericDate(now),
	}
	if ttl != 0 {
		c.ExpiresAt = gojwt.NewNumericDate(now.Add(ttl))
	}
	return c
}

// Sign serialises claims using the currently active key. The token carries a
// `kid` header so Verify can pick the right verification key.
func KeySetSign[C Claims](ks *KeySet, claims C) (string, error) {
	ks.mu.RLock()
	active := ks.active
	entry := ks.keys[active]
	ks.mu.RUnlock()
	if active == "" || entry == nil {
		return "", errors.New("jwt: no active key in key set")
	}
	if entry.signKey == nil {
		return "", fmt.Errorf("jwt: kid %q is verify-only", active)
	}
	tok := gojwt.NewWithClaims(signingMethodFor(entry.alg), claims)
	tok.Header["kid"] = active
	signed, err := tok.SignedString(entry.signKey)
	if err != nil {
		return "", fmt.Errorf("jwt: sign: %w", err)
	}
	return signed, nil
}

// KeySetVerify parses and verifies raw, picking the key by the token's `kid` header.
//
// Tokens without a `kid` header are rejected with ErrInvalid even when only
// one key is configured — kid is mandatory for the key-set API so rotation
// invariants stay consistent.
func KeySetVerify[C Claims](ks *KeySet, raw string) (C, error) {
	var zero C
	out := new(C)
	parsed, err := gojwt.ParseWithClaims(raw, anyAsClaims(out), func(t *gojwt.Token) (any, error) {
		kidAny, ok := t.Header["kid"]
		if !ok {
			return nil, errors.New("missing kid header")
		}
		kid, _ := kidAny.(string)
		ks.mu.RLock()
		entry, present := ks.keys[kid]
		ks.mu.RUnlock()
		if !present {
			return nil, fmt.Errorf("unknown kid %q", kid)
		}
		if t.Method.Alg() != string(entry.alg) {
			return nil, fmt.Errorf("kid %q expects %s, got %s", kid, entry.alg, t.Method.Alg())
		}
		return entry.verifyKey, nil
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

func signingMethodFor(alg Algorithm) gojwt.SigningMethod {
	switch alg {
	case RS256:
		return gojwt.SigningMethodRS256
	default:
		return gojwt.SigningMethodHS256
	}
}
