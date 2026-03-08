package tests_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AryanAg08/loginfy.go/core"
	"github.com/AryanAg08/loginfy.go/sessions/jwt"
)

func newJWTManager() *jwt.JWTSessionManager {
	return jwt.New(jwt.Config{Secret: "test-secret", Expiration: time.Hour})
}

func newTestContext() *core.Context {
	return &core.Context{
		Request:   httptest.NewRequest(http.MethodGet, "/", nil),
		Response:  httptest.NewRecorder(),
		RequestID: "test-req",
	}
}

func TestCreateSession(t *testing.T) {
	jm := newJWTManager()
	token, err := jm.CreateSession("user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	// JWT should have 3 parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}
}

func TestValidateSession(t *testing.T) {
	jm := newJWTManager()
	token, _ := jm.CreateSession("user-1")

	ctx := newTestContext()
	userID, err := jm.ValidateSession(ctx, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("expected user ID 'user-1', got %q", userID)
	}
}

func TestExpiredToken(t *testing.T) {
	jm := jwt.New(jwt.Config{Secret: "test-secret", Expiration: -time.Hour})
	token, _ := jm.CreateSession("user-1")

	ctx := newTestContext()
	_, err := jm.ValidateSession(ctx, token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestInvalidToken(t *testing.T) {
	jm := newJWTManager()
	ctx := newTestContext()

	_, err := jm.ValidateSession(ctx, "not-a-valid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestInvalidSignature(t *testing.T) {
	jm1 := jwt.New(jwt.Config{Secret: "secret-1", Expiration: time.Hour})
	jm2 := jwt.New(jwt.Config{Secret: "secret-2", Expiration: time.Hour})

	token, _ := jm1.CreateSession("user-1")

	ctx := newTestContext()
	_, err := jm2.ValidateSession(ctx, token)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestCreateSessionWithUser(t *testing.T) {
	jm := newJWTManager()
	user := &core.User{
		ID:    "user-1",
		Email: "test@test.com",
		Roles: []string{"admin"},
	}

	token, err := jm.CreateSessionWithUser(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Validate it returns the correct user ID
	ctx := newTestContext()
	userID, err := jm.ValidateSession(ctx, token)
	if err != nil {
		t.Fatalf("unexpected error validating: %v", err)
	}
	if userID != "user-1" {
		t.Fatalf("expected user ID 'user-1', got %q", userID)
	}
}
