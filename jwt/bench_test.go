package jwt

import (
	"testing"
	"time"
)

type benchClaims struct {
	RegisteredClaims
	UID  string `json:"uid"`
	Tags []string
}

func BenchmarkSign_HMAC(b *testing.B) {
	s := NewHMAC("iss", []byte("secret"), time.Hour)
	c := benchClaims{
		RegisteredClaims: s.NewClaims(),
		UID:              "u-1",
		Tags:             []string{"a", "b"},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Sign(s, c)
	}
}

func BenchmarkVerify_HMAC(b *testing.B) {
	s := NewHMAC("iss", []byte("secret"), time.Hour)
	tok, _ := Sign(s, benchClaims{
		RegisteredClaims: s.NewClaims(),
		UID:              "u-1",
	})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Verify[benchClaims](s, tok)
	}
}

func BenchmarkKeySetVerify(b *testing.B) {
	ks := NewKeySet("iss", time.Hour)
	_ = ks.AddHMACKey("k1", []byte("secret"))
	tok, _ := KeySetSign(ks, benchClaims{RegisteredClaims: ks.NewClaims(), UID: "u-1"})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = KeySetVerify[benchClaims](ks, tok)
	}
}
