package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

// GenerateAPIKey creates a new random API key.
// Returns the raw key (shown to user once) and its SHA-256 hex hash (stored in DB).
func GenerateAPIKey() (raw string, hash string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("auth: failed to generate random bytes: " + err.Error())
	}
	raw = base64.URLEncoding.EncodeToString(b)
	hash = HashAPIKey(raw)
	return raw, hash
}

// HashAPIKey returns the SHA-256 hex hash of a raw API key string.
func HashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
