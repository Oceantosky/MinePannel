package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const AuthTimeWindow = 60 // seconds

// GenerateMAC creates an HMAC-SHA256 signature for inter-node / PCL client auth.
func GenerateMAC(secret string, timestamp int64, method, path, bodyHash string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(fmt.Sprintf("%d:%s:%s:%s", timestamp, method, path, bodyHash)))
	return hex.EncodeToString(h.Sum(nil))
}

// ValidateMAC checks an HMAC-SHA256 token against the expected signature.
func ValidateMAC(token string, timestamp int64, secret, method, path, bodyHash string) bool {
	expected := GenerateMAC(secret, timestamp, method, path, bodyHash)
	return hmac.Equal([]byte(token), []byte(expected))
}

// Sha256Hash returns the raw SHA-256 hash bytes of a string.
func Sha256Hash(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}
