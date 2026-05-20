package objstore

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Live S3 calls are exercised via the examples/objstore_basic demo (needs a
// real endpoint and creds). These tests cover the pieces that don't need a
// network round-trip.

func TestOpen_RegionRequired(t *testing.T) {
	if _, err := Open(context.Background(), Options{}); err == nil {
		t.Error("expected error for missing region")
	}
}

func TestResolveBucket_Default(t *testing.T) {
	c := &Client{bucket: "fallback"}
	if got, err := c.resolveBucket(""); err != nil || got != "fallback" {
		t.Errorf("default failed: got=%q err=%v", got, err)
	}
	if got, err := c.resolveBucket("explicit"); err != nil || got != "explicit" {
		t.Errorf("override failed: got=%q err=%v", got, err)
	}
}

func TestResolveBucket_NoDefault(t *testing.T) {
	c := &Client{}
	if _, err := c.resolveBucket(""); err == nil {
		t.Error("expected error when no DefaultBucket and bucket arg empty")
	}
}

func TestIsNotFoundErr(t *testing.T) {
	if !isNotFoundErr(&types.NoSuchKey{}) {
		t.Error("NoSuchKey should be classified as not-found")
	}
	if !isNotFoundErr(&types.NotFound{}) {
		t.Error("NotFound should be classified as not-found")
	}
	if isNotFoundErr(errors.New("other")) {
		t.Error("arbitrary error should not be classified as not-found")
	}
	if isNotFoundErr(nil) {
		t.Error("nil should not be classified as not-found")
	}
}

func TestErrNotFoundExported(t *testing.T) {
	if ErrNotFound == nil || ErrNotFound.Error() == "" {
		t.Error("ErrNotFound should be a non-nil exported sentinel")
	}
}

func TestDerefHelpers(t *testing.T) {
	if derefStr(nil) != "" || derefInt64(nil) != 0 {
		t.Error("nil deref should return zero values")
	}
	s, n := "x", int64(42)
	if derefStr(&s) != "x" || derefInt64(&n) != 42 {
		t.Error("deref should unwrap")
	}
}
