package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

// Token is a per-boot secret. Restart rotates it.
type Token string

// NewToken generates 32 random bytes, hex-encoded.
func NewToken() (Token, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return Token(hex.EncodeToString(b[:])), nil
}

// Bearer extracts the token from `?token=` or `Authorization: Bearer`.
func Bearer(queryToken, authHeader string) string {
	if queryToken != "" {
		return queryToken
	}
	if rest, ok := strings.CutPrefix(authHeader, "Bearer "); ok {
		return rest
	}
	return ""
}

// Check compares in constant time.
func (t Token) Check(got string) bool {
	return subtle.ConstantTimeCompare([]byte(string(t)), []byte(got)) == 1
}
