package games

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
)

type practiceCursor struct {
	GameID      uuid.UUID `json:"g"`
	RoundNumber int       `json:"r"`
}

func derivePracticeCursorKey(secret string) []byte {
	sum := sha256.Sum256([]byte("geoguess:practice-cursor:v1:" + secret))
	return sum[:]
}

func encodePracticeCursor(key []byte, gameID uuid.UUID, roundNumber int) string {
	payload, _ := json.Marshal(practiceCursor{GameID: gameID, RoundNumber: roundNumber})
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	signature := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func decodePracticeCursor(key []byte, value string, gameID uuid.UUID) (int, error) {
	if value == "" {
		return 0, nil
	}
	if len(key) == 0 {
		return 0, ErrInvalidCursor
	}
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return 0, ErrInvalidCursor
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, ErrInvalidCursor
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, ErrInvalidCursor
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return 0, ErrInvalidCursor
	}
	var cursor practiceCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.GameID != gameID || cursor.RoundNumber < 0 {
		return 0, ErrInvalidCursor
	}
	return cursor.RoundNumber, nil
}
