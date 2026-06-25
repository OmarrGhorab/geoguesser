package auth_test

import (
	"testing"

	"github.com/raven/geoguess/backend/internal/auth"
)

func TestBCryptHasherHashesAndVerifiesPassword(t *testing.T) {
	hasher := auth.NewBCryptHasherWithCost(4) // low cost for fast tests
	password := "correct horse battery staple"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}
	if hash == "" {
		t.Fatal("hash must not be empty")
	}
	if hash == password {
		t.Fatal("hash must not equal plaintext")
	}

	if err := hasher.Verify(password, hash); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
}

func TestBCryptHasherRejectsWrongPassword(t *testing.T) {
	hasher := auth.NewBCryptHasherWithCost(4)
	password := "correct horse battery staple"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("hash failed: %v", err)
	}

	if err := hasher.Verify("wrong password", hash); err == nil {
		t.Fatal("expected verification to fail for wrong password")
	}
}

func TestBCryptHasherRejectsEmptyPassword(t *testing.T) {
	hasher := auth.NewBCryptHasher()
	if _, err := hasher.Hash(""); err == nil {
		t.Fatal("expected error for empty password")
	}
}
