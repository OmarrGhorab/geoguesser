package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	otpKeyPrefix   = "password_reset_otp"
	otpRatePrefix  = "password_reset_rate"
	otpLength      = 6
	otpMaxAttempts = 3
	otpRateLimit   = 3
	otpRateWindow  = 10 * time.Minute
)

// OTPStore manages password reset OTPs in Redis.
type OTPStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewOTPStore returns a new OTP store.
func NewOTPStore(client *redis.Client, ttl time.Duration) *OTPStore {
	return &OTPStore{client: client, ttl: ttl}
}

// Generate creates a new OTP for the given email and stores it in Redis.
func (o *OTPStore) Generate(ctx context.Context, email string) (string, error) {
	code, err := generateNumericCode(otpLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate otp: %w", err)
	}

	key := otpKey(otpKeyPrefix, email)
	pipe := o.client.Pipeline()
	pipe.Set(ctx, key, code, o.ttl)
	pipe.Set(ctx, fmt.Sprintf("%s:attempts", key), 0, o.ttl)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to store otp: %w", err)
	}
	return code, nil
}

// Validate checks the OTP for the given email and increments the attempt counter.
func (o *OTPStore) Validate(ctx context.Context, email, code string) (bool, error) {
	key := otpKey(otpKeyPrefix, email)

	stored, err := o.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to read otp: %w", err)
	}

	attempts, err := o.client.Incr(ctx, fmt.Sprintf("%s:attempts", key)).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment otp attempts: %w", err)
	}
	if attempts > otpMaxAttempts {
		_ = o.client.Del(ctx, key, fmt.Sprintf("%s:attempts", key))
		return false, nil
	}

	if stored != code {
		return false, nil
	}

	_ = o.client.Del(ctx, key, fmt.Sprintf("%s:attempts", key))
	return true, nil
}

// Delete removes the OTP for the given email.
func (o *OTPStore) Delete(ctx context.Context, email string) error {
	key := otpKey(otpKeyPrefix, email)
	return o.client.Del(ctx, key, fmt.Sprintf("%s:attempts", key)).Err()
}

// AllowRequest checks whether a new OTP request is within the rate limit.
func (o *OTPStore) AllowRequest(ctx context.Context, email string) (bool, error) {
	key := otpKey(otpRatePrefix, email)
	count, err := o.client.Incr(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to increment otp rate limit: %w", err)
	}
	if count == 1 {
		_ = o.client.Expire(ctx, key, otpRateWindow)
	}
	return count <= otpRateLimit, nil
}

func otpKey(prefix, email string) string {
	normalized := strings.ToLower(strings.TrimSpace(email))
	sum := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%s:%s", prefix, hex.EncodeToString(sum[:]))
}

func generateNumericCode(length int) (string, error) {
	const digits = "0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b), nil
}
