package tests_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"time"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/sessions/jwt"
)

// mockStrategy implements core.Strategy for testing
type mockStrategy struct {
	name string
	user *core.User
	err  error
}

func (m *mockStrategy) Name() string { return m.name }
func (m *mockStrategy) Authenticate(ctx *core.Context) (*core.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}

func TestNewLoginfy(t *testing.T) {
	l := core.New()
	if l == nil {
		t.Fatal("expected non-nil Loginfy instance")
	}

	s := &mockStrategy{name: "mock"}
	l.Use(s)

	got, ok := l.GetStrategy("mock")
	if !ok {
		t.Fatal("expected strategy to be registered")
	}
	if got.Name() != "mock" {
		t.Fatalf("expected strategy name 'mock', got %q", got.Name())
	}
}

func TestAuthenticate(t *testing.T) {
	l := core.New()
	user := &core.User{ID: "u1", Email: "a@b.com", Roles: []string{"user"}}
	l.Use(&mockStrategy{name: "mock", user: user})

	ctx := &core.Context{
		Request:   httptest.NewRequest(http.MethodGet, "/", nil),
		Response:  httptest.NewRecorder(),
		Loginfy:   l,
		RequestID: "req-1",
	}

	got, err := l.Authenticate("mock", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "u1" {
		t.Fatalf("expected user ID 'u1', got %q", got.ID)
	}

	// user should be stored in context
	ctxUser, ok := ctx.GetUser()
	if !ok || ctxUser.ID != "u1" {
		t.Fatal("expected user to be stored in context")
	}
}

func TestAuthenticateUnknownStrategy(t *testing.T) {
	l := core.New()
	ctx := &core.Context{
		Request:   httptest.NewRequest(http.MethodGet, "/", nil),
		Response:  httptest.NewRecorder(),
		Loginfy:   l,
		RequestID: "req-2",
	}

	_, err := l.Authenticate("nonexistent", ctx)
	if err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}

func TestLoginLogout(t *testing.T) {
	l := core.New()
	jwtManager := jwt.New(jwt.Config{Secret: "test-secret", Expiration: time.Hour})
	l.SetSessionManager(jwtManager)

	user := &core.User{ID: "u1", Email: "a@b.com"}

	token, err := l.Login(user)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	ctx := &core.Context{
		Request:   httptest.NewRequest(http.MethodGet, "/", nil),
		Response:  httptest.NewRecorder(),
		Loginfy:   l,
		RequestID: "req-3",
	}

	err = l.Logout(ctx, token)
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}
}

func TestLoginNoSessionManager(t *testing.T) {
	l := core.New()
	user := &core.User{ID: "u1"}
	_, err := l.Login(user)
	if err == nil {
		t.Fatal("expected error when no session manager set")
	}
}

func TestContext(t *testing.T) {
	ctx := &core.Context{
		Request:   httptest.NewRequest(http.MethodGet, "/", nil),
		Response:  httptest.NewRecorder(),
		RequestID: "req-ctx",
	}

	// Test Set/Get
	ctx.Set("key1", "value1")
	val, ok := ctx.Get("key1")
	if !ok || val != "value1" {
		t.Fatal("expected to get 'value1'")
	}

	// Test GetString
	ctx.Set("strKey", "hello")
	if ctx.GetString("strKey") != "hello" {
		t.Fatal("expected GetString to return 'hello'")
	}
	if ctx.GetString("missing") != "" {
		t.Fatal("expected empty string for missing key")
	}

	// Test GetString with non-string value
	ctx.Set("intKey", 42)
	if ctx.GetString("intKey") != "" {
		t.Fatal("expected empty string for non-string value")
	}

	// Test SetUser/GetUser
	user := &core.User{ID: "u1", Email: "test@test.com", Roles: []string{"admin"}}
	ctx.SetUser(user)
	gotUser, ok := ctx.GetUser()
	if !ok || gotUser.ID != "u1" {
		t.Fatal("expected to retrieve user from context")
	}

	// Test HasUser
	if !ctx.HasUser() {
		t.Fatal("expected HasUser to return true")
	}

	// Test HasUser on empty context
	emptyCtx := &core.Context{
		Request:   httptest.NewRequest(http.MethodGet, "/", nil),
		Response:  httptest.NewRecorder(),
		RequestID: "req-empty",
	}
	if emptyCtx.HasUser() {
		t.Fatal("expected HasUser to return false on empty context")
	}
}

func TestUser(t *testing.T) {
	user := &core.User{
		ID:    "u1",
		Email: "test@test.com",
		Roles: []string{"admin", "editor"},
	}

	if !user.HasRole("admin") {
		t.Fatal("expected user to have 'admin' role")
	}
	if user.HasRole("superadmin") {
		t.Fatal("expected user to NOT have 'superadmin' role")
	}
	if !user.HasAnyRole("viewer", "admin") {
		t.Fatal("expected HasAnyRole to return true")
	}
	if user.HasAnyRole("viewer", "superadmin") {
		t.Fatal("expected HasAnyRole to return false")
	}

	// Test nil roles
	nilRolesUser := &core.User{ID: "u2"}
	if nilRolesUser.HasRole("admin") {
		t.Fatal("expected HasRole to return false for nil roles")
	}
	if nilRolesUser.HasAnyRole("admin") {
		t.Fatal("expected HasAnyRole to return false for nil roles")
	}
}

func TestHooks(t *testing.T) {
	l := core.New()
	user := &core.User{ID: "u1", Email: "a@b.com"}
	l.Use(&mockStrategy{name: "mock", user: user})

	loginCalled := false
	l.SetHooks(core.Hooks{
		OnLogin: func(u *core.User) {
			loginCalled = true
			if u.ID != "u1" {
				t.Fatalf("expected user ID 'u1' in hook, got %q", u.ID)
			}
		},
	})

	ctx := &core.Context{
		Request:   httptest.NewRequest(http.MethodGet, "/", nil),
		Response:  httptest.NewRecorder(),
		Loginfy:   l,
		RequestID: "req-hooks",
	}

	_, err := l.Authenticate("mock", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !loginCalled {
		t.Fatal("expected OnLogin hook to be called")
	}
}

func TestContextWithLoginfy(t *testing.T) {
	l := core.New()
	loginfyCtx := &core.Context{
		Request:   httptest.NewRequest(http.MethodGet, "/", nil),
		Response:  httptest.NewRecorder(),
		Loginfy:   l,
		RequestID: "req-with",
	}

	goCtx := core.ContextWithLoginfy(context.Background(), loginfyCtx)
	retrieved, ok := core.LoginfyFromContext(goCtx)
	if !ok {
		t.Fatal("expected to retrieve Loginfy context")
	}
	if retrieved.RequestID != "req-with" {
		t.Fatalf("expected RequestID 'req-with', got %q", retrieved.RequestID)
	}
}

func TestLoginfyFromContextMissing(t *testing.T) {
	_, ok := core.LoginfyFromContext(context.Background())
	if ok {
		t.Fatal("expected LoginfyFromContext to return false for empty context")
	}
}
