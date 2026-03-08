package tests_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/middleware"
	"github.com/AryanAg08/loginfy-go/sessions/jwt"
	"github.com/AryanAg08/loginfy-go/storage/memory"
)

func TestRequireAuthMissingToken(t *testing.T) {
	handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuthWithToken(t *testing.T) {
	handler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireAuthWithLoginfy(t *testing.T) {
	l := core.New()
	store := memory.New()
	jm := jwt.New(jwt.Config{Secret: "test-secret", Expiration: time.Hour})
	l.SetStorage(store)
	l.SetSessionManager(jm)

	// Create a user
	store.CreateUser(&core.User{ID: "u1", Email: "test@test.com", Roles: []string{"user"}})

	// Create a session token
	token, _ := l.Login(&core.User{ID: "u1"})

	// Build handler chain: Mount -> RequireAuthWithLoginfy -> handler
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loginfyCtx, ok := core.LoginfyFromContext(r.Context())
		if !ok {
			t.Fatal("expected loginfy context")
		}
		user, ok := loginfyCtx.GetUser()
		if !ok {
			t.Fatal("expected user in context")
		}
		if user.ID != "u1" {
			t.Fatalf("expected user ID 'u1', got %q", user.ID)
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := l.Mount()(middleware.RequireAuthWithLoginfy(l)(innerHandler))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireRoleAuthorized(t *testing.T) {
	l := core.New()
	user := &core.User{ID: "u1", Email: "admin@test.com", Roles: []string{"admin"}}

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequireRole(l, "admin")(innerHandler)

	// Create request with loginfy context containing user
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	loginfyCtx := &core.Context{
		Request:   req,
		Response:  rec,
		Loginfy:   l,
		RequestID: "req-role",
	}
	loginfyCtx.SetUser(user)
	req = req.WithContext(core.ContextWithLoginfy(req.Context(), loginfyCtx))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireRoleUnauthorized(t *testing.T) {
	l := core.New()
	user := &core.User{ID: "u1", Email: "user@test.com", Roles: []string{"user"}}

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequireRole(l, "admin")(innerHandler)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()

	loginfyCtx := &core.Context{
		Request:   req,
		Response:  rec,
		Loginfy:   l,
		RequestID: "req-role-unauth",
	}
	loginfyCtx.SetUser(user)
	req = req.WithContext(core.ContextWithLoginfy(req.Context(), loginfyCtx))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestRequirePermissionWithPermission(t *testing.T) {
	l := core.New()
	user := &core.User{
		ID:    "u1",
		Email: "test@test.com",
		Metadata: map[string]interface{}{
			"permissions": []string{"read", "write"},
		},
	}

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequirePermission(l, "read")(innerHandler)

	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	rec := httptest.NewRecorder()

	loginfyCtx := &core.Context{
		Request:   req,
		Response:  rec,
		Loginfy:   l,
		RequestID: "req-perm",
	}
	loginfyCtx.SetUser(user)
	req = req.WithContext(core.ContextWithLoginfy(req.Context(), loginfyCtx))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequirePermissionWithoutPermission(t *testing.T) {
	l := core.New()
	user := &core.User{
		ID:    "u1",
		Email: "test@test.com",
		Metadata: map[string]interface{}{
			"permissions": []string{"read"},
		},
	}

	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware.RequirePermission(l, "delete")(innerHandler)

	req := httptest.NewRequest(http.MethodDelete, "/data", nil)
	rec := httptest.NewRecorder()

	loginfyCtx := &core.Context{
		Request:   req,
		Response:  rec,
		Loginfy:   l,
		RequestID: "req-perm-denied",
	}
	loginfyCtx.SetUser(user)
	req = req.WithContext(core.ContextWithLoginfy(req.Context(), loginfyCtx))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
