package jwt_test

import (
	"errors"
	"fmt"
	"time"

	"common/jwt"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"uid"`
}

// ExampleNewHMAC issues and verifies a JWT using HS256.
func ExampleNewHMAC() {
	s := jwt.NewHMAC("billing-iss", []byte("super-secret"), time.Hour)
	tok, _ := jwt.Sign(s, Claims{
		RegisteredClaims: s.NewClaims(),
		UserID:           "u-1",
	})
	c, _ := jwt.Verify[Claims](s, tok)
	fmt.Println("uid:", c.UserID)
	// Output:
	// uid: u-1
}

// ExampleNewKeySet rotates from key k1 to k2 without invalidating in-flight
// tokens, then retires k1 once they have all expired.
func ExampleNewKeySet() {
	ks := jwt.NewKeySet("iss", time.Hour)
	_ = ks.AddHMACKey("k1", []byte("old"))
	_ = ks.AddHMACKey("k2", []byte("new"))

	// While k1 is still active, sign with it:
	old, _ := jwt.KeySetSign(ks, Claims{RegisteredClaims: ks.NewClaims(), UserID: "u-1"})

	_ = ks.SetActive("k2") // flip to new key for issuance.
	new, _ := jwt.KeySetSign(ks, Claims{RegisteredClaims: ks.NewClaims(), UserID: "u-2"})

	// Old token still verifies because k1 is still in the set.
	_, errOld := jwt.KeySetVerify[Claims](ks, old)
	_, errNew := jwt.KeySetVerify[Claims](ks, new)
	fmt.Println("old verifies:", errOld == nil)
	fmt.Println("new verifies:", errNew == nil)

	ks.Remove("k1") // retire k1 after every old token has expired.
	_, errAfter := jwt.KeySetVerify[Claims](ks, old)
	fmt.Println("after Remove(k1):", errors.Is(errAfter, jwt.ErrInvalid))
	// Output:
	// old verifies: true
	// new verifies: true
	// after Remove(k1): true
}
