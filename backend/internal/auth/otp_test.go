package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/raven/geoguess/backend/internal/auth"
	redisplatform "github.com/raven/geoguess/backend/internal/platform/redis"
)

func TestOTPStoreGenerateAndValidate(t *testing.T) {
	client, err := redisplatform.Open(context.Background(), "redis://localhost:6379/15")
	if err != nil {
		t.Skip("redis not available")
	}
	defer func() { _ = client.Close() }()

	store := auth.NewOTPStore(client, 5*time.Minute)
	email := "otp-test@example.com"

	code, err := store.Generate(context.Background(), email)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("otp length = %d, want 6", len(code))
	}

	valid, err := store.Validate(context.Background(), email, code)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if !valid {
		t.Fatal("expected otp to be valid")
	}

	valid, err = store.Validate(context.Background(), email, code)
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if valid {
		t.Fatal("expected otp to be invalid after use")
	}
}

func TestOTPStoreRejectsWrongCode(t *testing.T) {
	client, err := redisplatform.Open(context.Background(), "redis://localhost:6379/15")
	if err != nil {
		t.Skip("redis not available")
	}
	defer func() { _ = client.Close() }()

	store := auth.NewOTPStore(client, 5*time.Minute)
	email := "otp-wrong@example.com"

	if _, err := store.Generate(context.Background(), email); err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	valid, err := store.Validate(context.Background(), email, "000000")
	if err != nil {
		t.Fatalf("validate failed: %v", err)
	}
	if valid {
		t.Fatal("expected wrong otp to be rejected")
	}
}
