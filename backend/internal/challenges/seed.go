package challenges

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const sharedCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func DailyWindow(now time.Time, resetHourUTC int) (time.Time, time.Time, time.Time) {
	if resetHourUTC < 0 || resetHourUTC > 23 {
		resetHourUTC = 0
	}
	utc := now.UTC()
	start := time.Date(utc.Year(), utc.Month(), utc.Day(), resetHourUTC, 0, 0, 0, time.UTC)
	if utc.Before(start) {
		start = start.AddDate(0, 0, -1)
	}
	end := start.AddDate(0, 0, 1)
	date := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	return date, start, end
}

func DailySeed(date time.Time) string {
	sum := sha256.Sum256([]byte("geoguess:daily:" + date.UTC().Format("2006-01-02")))
	return hex.EncodeToString(sum[:8])
}

func SharedSeed() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate shared seed: %w", err)
	}
	return strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:]), "="), nil
}

func SharedCode() (string, error) {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate shared code: %w", err)
	}
	out := make([]byte, 10)
	for i := range b {
		out[i] = sharedCodeAlphabet[int(b[i])%len(sharedCodeAlphabet)]
	}
	return string(out), nil
}
