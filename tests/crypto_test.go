package tests_test

import (
	"strings"
	"testing"

	"github.com/AryanAg08/loginfy-go/pkg/crypto"
)

func TestHashPassword(t *testing.T) {
	hash, err := crypto.HashPassword("mypassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == "mypassword" {
		t.Fatal("hash should not equal the plain password")
	}
}

func TestVerifyPassword(t *testing.T) {
	hash, _ := crypto.HashPassword("mypassword")
	err := crypto.VerifyPassword("mypassword", hash)
	if err != nil {
		t.Fatalf("expected password to verify successfully: %v", err)
	}
}

func TestVerifyWrongPassword(t *testing.T) {
	hash, _ := crypto.HashPassword("mypassword")
	err := crypto.VerifyPassword("wrongpassword", hash)
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestGenerateToken(t *testing.T) {
	token, err := crypto.GenerateToken(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Generate two tokens; they should be different
	token2, _ := crypto.GenerateToken(32)
	if token == token2 {
		t.Fatal("expected two generated tokens to be different")
	}
}

func TestGenerateAPIKey(t *testing.T) {
	key, err := crypto.GenerateAPIKey("lgnfy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(key, "lgnfy_") {
		t.Fatalf("expected API key to start with 'lgnfy_', got %q", key)
	}

	// Without prefix
	key2, err := crypto.GenerateAPIKey("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(key2, "_") {
		t.Fatalf("expected no prefix separator, got %q", key2)
	}
}

func TestConstantTimeCompare(t *testing.T) {
	if !crypto.ConstantTimeCompare("abc", "abc") {
		t.Fatal("expected equal strings to match")
	}
	if crypto.ConstantTimeCompare("abc", "xyz") {
		t.Fatal("expected different strings to not match")
	}
	if crypto.ConstantTimeCompare("abc", "abcd") {
		t.Fatal("expected different length strings to not match")
	}
}
