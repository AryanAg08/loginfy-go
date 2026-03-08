package tests_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AryanAg08/loginfy.go/core"
	"github.com/AryanAg08/loginfy.go/pkg/crypto"
	"github.com/AryanAg08/loginfy.go/storage/memory"
	"github.com/AryanAg08/loginfy.go/strategies/emailPassword"
)

func setupStrategyTest(t *testing.T) (*core.Loginfy, *memory.MemoryStorage) {
	t.Helper()
	l := core.New()
	store := memory.New()
	l.SetStorage(store)

	strategy := emailPassword.New()
	l.Use(strategy)

	return l, store
}

func TestEmailPasswordAuthenticate(t *testing.T) {
	l, store := setupStrategyTest(t)

	hashed, err := crypto.HashPassword("password123")
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}
	store.CreateUser(&core.User{ID: "u1", Email: "test@test.com", Password: hashed})

	ctx := &core.Context{
		Request:   httptest.NewRequest(http.MethodPost, "/login", nil),
		Response:  httptest.NewRecorder(),
		Loginfy:   l,
		RequestID: "req-auth",
	}
	ctx.Set("email", "test@test.com")
	ctx.Set("password", "password123")

	user, err := l.Authenticate("email_password", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "u1" {
		t.Fatalf("expected user ID 'u1', got %q", user.ID)
	}
}

func TestEmailPasswordRegister(t *testing.T) {
	l, _ := setupStrategyTest(t)
	strategy, _ := l.GetStrategy("email_password")
	epStrategy := strategy.(*emailPassword.EmailPasswordStrategy)

	ctx := &core.Context{
		Request:   httptest.NewRequest(http.MethodPost, "/register", nil),
		Response:  httptest.NewRecorder(),
		Loginfy:   l,
		RequestID: "req-register",
	}
	ctx.Set("email", "new@test.com")
	ctx.Set("password", "newpassword")

	user, err := epStrategy.Register(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "new@test.com" {
		t.Fatalf("expected email 'new@test.com', got %q", user.Email)
	}
	if user.ID == "" {
		t.Fatal("expected non-empty user ID")
	}
}

func TestMissingCredentials(t *testing.T) {
	l, _ := setupStrategyTest(t)

	// Missing both
	ctx := &core.Context{
		Request:   httptest.NewRequest(http.MethodPost, "/login", nil),
		Response:  httptest.NewRecorder(),
		Loginfy:   l,
		RequestID: "req-missing",
	}
	_, err := l.Authenticate("email_password", ctx)
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}

	// Missing password
	ctx2 := &core.Context{
		Request:   httptest.NewRequest(http.MethodPost, "/login", nil),
		Response:  httptest.NewRecorder(),
		Loginfy:   l,
		RequestID: "req-missing-pw",
	}
	ctx2.Set("email", "test@test.com")
	_, err = l.Authenticate("email_password", ctx2)
	if err == nil {
		t.Fatal("expected error for missing password")
	}
}

func TestInvalidPassword(t *testing.T) {
	l, store := setupStrategyTest(t)

	hashed, _ := crypto.HashPassword("correctpassword")
	store.CreateUser(&core.User{ID: "u1", Email: "test@test.com", Password: hashed})

	ctx := &core.Context{
		Request:   httptest.NewRequest(http.MethodPost, "/login", nil),
		Response:  httptest.NewRecorder(),
		Loginfy:   l,
		RequestID: "req-badpw",
	}
	ctx.Set("email", "test@test.com")
	ctx.Set("password", "wrongpassword")

	_, err := l.Authenticate("email_password", ctx)
	if err == nil {
		t.Fatal("expected error for invalid password")
	}
}

func TestNoStorage(t *testing.T) {
	l := core.New()
	l.Use(emailPassword.New())

	ctx := &core.Context{
		Request:   httptest.NewRequest(http.MethodPost, "/login", nil),
		Response:  httptest.NewRecorder(),
		Loginfy:   l,
		RequestID: "req-nostorage",
	}
	ctx.Set("email", "test@test.com")
	ctx.Set("password", "password123")

	_, err := l.Authenticate("email_password", ctx)
	if err == nil {
		t.Fatal("expected error when storage is not configured")
	}
}
