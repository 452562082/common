package jwt

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestKeySet_HMAC_SignVerify(t *testing.T) {
	ks := NewKeySet("iss", time.Hour)
	if err := ks.AddHMACKey("k1", []byte("secret-1")); err != nil {
		t.Fatal(err)
	}

	tok, err := KeySetSign(ks, myClaims{
		RegisteredClaims: ks.NewClaims(),
		UID:              "u-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := KeySetVerify[myClaims](ks, tok)
	if err != nil {
		t.Fatal(err)
	}
	if out.UID != "u-1" {
		t.Errorf("UID = %q", out.UID)
	}
}

func TestKeySet_Rotation(t *testing.T) {
	ks := NewKeySet("iss", time.Hour)
	_ = ks.AddHMACKey("k1", []byte("old"))
	_ = ks.AddHMACKey("k2", []byte("new"))

	// Default active is the first one we added (k1).
	if ks.ActiveKID() != "k1" {
		t.Fatalf("active = %q, want k1", ks.ActiveKID())
	}
	oldTok, _ := KeySetSign(ks, myClaims{RegisteredClaims: ks.NewClaims(), UID: "old"})

	if err := ks.SetActive("k2"); err != nil {
		t.Fatal(err)
	}
	newTok, _ := KeySetSign(ks, myClaims{RegisteredClaims: ks.NewClaims(), UID: "new"})

	// Both tokens still verify while both keys are present.
	if _, err := KeySetVerify[myClaims](ks, oldTok); err != nil {
		t.Errorf("old token should still verify: %v", err)
	}
	if _, err := KeySetVerify[myClaims](ks, newTok); err != nil {
		t.Errorf("new token should verify: %v", err)
	}

	// Retire k1; the old token now fails.
	ks.Remove("k1")
	if _, err := KeySetVerify[myClaims](ks, oldTok); !errors.Is(err, ErrInvalid) {
		t.Errorf("expected ErrInvalid after kid removed; got %v", err)
	}
}

func TestKeySet_MissingKidRejected(t *testing.T) {
	ks := NewKeySet("iss", time.Hour)
	_ = ks.AddHMACKey("k1", []byte("s"))

	// Hand-roll a token without kid (use the old single-key API).
	s := NewHMAC("iss", []byte("s"), time.Hour)
	plain, _ := Sign(s, myClaims{RegisteredClaims: s.NewClaims()})

	if _, err := KeySetVerify[myClaims](ks, plain); !errors.Is(err, ErrInvalid) {
		t.Errorf("kid-less token must be rejected; got %v", err)
	}
}

func TestKeySet_UnknownKidRejected(t *testing.T) {
	ks1 := NewKeySet("iss", time.Hour)
	_ = ks1.AddHMACKey("k1", []byte("s"))
	tok, _ := KeySetSign(ks1, myClaims{RegisteredClaims: ks1.NewClaims()})

	ks2 := NewKeySet("iss", time.Hour)
	_ = ks2.AddHMACKey("k99", []byte("other"))

	if _, err := KeySetVerify[myClaims](ks2, tok); !errors.Is(err, ErrInvalid) {
		t.Errorf("token signed by unknown kid must be rejected: %v", err)
	}
}

func TestKeySet_AlgMismatchRejected(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	rsaKS := NewKeySet("iss", time.Hour)
	_ = rsaKS.AddRSAKey("rsa1", priv, &priv.PublicKey)
	rsaTok, _ := KeySetSign(rsaKS, myClaims{RegisteredClaims: rsaKS.NewClaims()})

	hmacKS := NewKeySet("iss", time.Hour)
	_ = hmacKS.AddHMACKey("rsa1", []byte("secret")) // same kid, different alg!

	if _, err := KeySetVerify[myClaims](hmacKS, rsaTok); !errors.Is(err, ErrInvalid) {
		t.Errorf("alg mismatch must be rejected: %v", err)
	}
}

func TestKeySet_SetActiveUnknownErrors(t *testing.T) {
	ks := NewKeySet("iss", time.Hour)
	if err := ks.SetActive("nope"); err == nil {
		t.Error("expected error setting unknown kid active")
	}
}

func TestKeySet_VerifyOnlyKey(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)

	// Issuer side: full keypair.
	issuer := NewKeySet("issuer", time.Hour)
	_ = issuer.AddRSAKey("a", priv, &priv.PublicKey)
	tok, _ := KeySetSign(issuer, myClaims{RegisteredClaims: issuer.NewClaims(), UID: "u"})

	// Verifier side: only the public key, no signing key.
	verifier := NewKeySet("issuer", time.Hour)
	_ = verifier.AddVerifyOnlyRSAKey("a", &priv.PublicKey)
	if _, err := KeySetVerify[myClaims](verifier, tok); err != nil {
		t.Errorf("verify-only key path: %v", err)
	}

	// And signing must error — either because there's no active key, or
	// because the active one has no signKey. Either is acceptable.
	if _, err := KeySetSign(verifier, myClaims{RegisteredClaims: verifier.NewClaims()}); err == nil {
		t.Error("expected error: cannot sign with a verify-only key set")
	}
}

func TestKeySet_KidPresentInHeader(t *testing.T) {
	ks := NewKeySet("iss", time.Hour)
	_ = ks.AddHMACKey("k-1", []byte("s"))
	tok, _ := KeySetSign(ks, myClaims{RegisteredClaims: ks.NewClaims()})

	// Decode the JWT header to confirm kid is set. JWT header is the first
	// dot-separated segment; we can verify presence via the underlying parser.
	_, err := KeySetVerify[myClaims](ks, tok)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !strings.Contains(tok, ".") {
		t.Error("invalid jwt shape")
	}
}
