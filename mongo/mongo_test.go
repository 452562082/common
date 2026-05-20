package mongo

import (
	"context"
	"errors"
	"testing"

	mgo "go.mongodb.org/mongo-driver/v2/mongo"
)

// Live mongo calls are covered via an integration test environment; here we
// exercise the validation and sentinel-error surface.

func TestOpen_RequiresURI(t *testing.T) {
	if _, err := Open(context.Background(), Options{Database: "x"}); err == nil {
		t.Error("expected error for missing URI")
	}
}

func TestOpen_RequiresDatabase(t *testing.T) {
	if _, err := Open(context.Background(), Options{URI: "mongodb://localhost:27017"}); err == nil {
		t.Error("expected error for missing Database")
	}
}

func TestIsNoDocuments(t *testing.T) {
	if !IsNoDocuments(mgo.ErrNoDocuments) {
		t.Error("ErrNoDocuments should be recognised")
	}
	if IsNoDocuments(errors.New("other")) {
		t.Error("arbitrary error should not match")
	}
	if IsNoDocuments(nil) {
		t.Error("nil should not match")
	}
}

func TestIsDuplicateKey(t *testing.T) {
	// IsDuplicateKey returns false for plain errors; the live behaviour with a
	// real driver error is covered by the upstream driver's own tests.
	if IsDuplicateKey(errors.New("plain")) {
		t.Error("arbitrary error should not be classified as duplicate-key")
	}
	if IsDuplicateKey(nil) {
		t.Error("nil should not be classified as duplicate-key")
	}
}
