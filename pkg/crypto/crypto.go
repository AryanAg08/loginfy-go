package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrHashFailed  = errors.New("failed to hash password")
	ErrInvalidHash = errors.New("invalid password hash format")
	ErrMismatch    = errors.New("password does not match")
)

const DefaultCost = bcrypt.DefaultCost

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), DefaultCost)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrHashFailed, err)
	}
	return string(bytes), nil
}

// VerifyPassword compares a password with its hash
func VerifyPassword(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return ErrMismatch
	}
	return nil
}

// GenerateToken generates a cryptographically secure random token
func GenerateToken(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ConstantTimeCompare compares two strings in constant time to prevent timing attacks
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// GenerateAPIKey generates a prefixed API key (e.g., "lgnfy_abc123...")
func GenerateAPIKey(prefix string) (string, error) {
	token, err := GenerateToken(32)
	if err != nil {
		return "", err
	}
	// Remove padding
	token = strings.TrimRight(token, "=")
	if prefix != "" {
		return prefix + "_" + token, nil
	}
	return token, nil
}
