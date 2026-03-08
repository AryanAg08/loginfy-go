package tests_test

import (
	"testing"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/storage/memory"
)

func TestCreateUser(t *testing.T) {
	store := memory.New()
	user := &core.User{ID: "u1", Email: "test@test.com", Password: "hashed"}

	err := store.CreateUser(user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if store.Count() != 1 {
		t.Fatalf("expected count 1, got %d", store.Count())
	}
}

func TestGetUserByEmail(t *testing.T) {
	store := memory.New()
	user := &core.User{ID: "u1", Email: "test@test.com", Password: "hashed"}
	store.CreateUser(user)

	got, err := store.GetUserByEmail("test@test.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != "u1" {
		t.Fatalf("expected ID 'u1', got %q", got.ID)
	}

	_, err = store.GetUserByEmail("missing@test.com")
	if err == nil {
		t.Fatal("expected error for missing email")
	}
}

func TestGetUserById(t *testing.T) {
	store := memory.New()
	user := &core.User{ID: "u1", Email: "test@test.com", Password: "hashed"}
	store.CreateUser(user)

	got, err := store.GetUserById("u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Email != "test@test.com" {
		t.Fatalf("expected email 'test@test.com', got %q", got.Email)
	}

	_, err = store.GetUserById("missing")
	if err == nil {
		t.Fatal("expected error for missing ID")
	}
}

func TestUpdateUser(t *testing.T) {
	store := memory.New()
	user := &core.User{ID: "u1", Email: "old@test.com", Password: "hashed"}
	store.CreateUser(user)

	updated := &core.User{ID: "u1", Email: "new@test.com", Password: "hashed"}
	err := store.UpdateUser(updated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := store.GetUserByEmail("new@test.com")
	if got.ID != "u1" {
		t.Fatalf("expected ID 'u1', got %q", got.ID)
	}

	// Old email should not work
	_, err = store.GetUserByEmail("old@test.com")
	if err == nil {
		t.Fatal("expected error for old email after update")
	}

	// Update non-existent user
	err = store.UpdateUser(&core.User{ID: "missing", Email: "x@x.com"})
	if err == nil {
		t.Fatal("expected error for updating non-existent user")
	}
}

func TestDeleteUser(t *testing.T) {
	store := memory.New()
	user := &core.User{ID: "u1", Email: "test@test.com", Password: "hashed"}
	store.CreateUser(user)

	err := store.DeleteUser("u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.Count() != 0 {
		t.Fatalf("expected count 0, got %d", store.Count())
	}

	// Delete non-existent user
	err = store.DeleteUser("missing")
	if err == nil {
		t.Fatal("expected error for deleting non-existent user")
	}
}

func TestDuplicateUser(t *testing.T) {
	store := memory.New()
	user := &core.User{ID: "u1", Email: "test@test.com", Password: "hashed"}
	store.CreateUser(user)

	// Duplicate ID
	err := store.CreateUser(&core.User{ID: "u1", Email: "other@test.com"})
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}

	// Duplicate email
	err = store.CreateUser(&core.User{ID: "u2", Email: "test@test.com"})
	if err == nil {
		t.Fatal("expected error for duplicate email")
	}
}

func TestClear(t *testing.T) {
	store := memory.New()
	store.CreateUser(&core.User{ID: "u1", Email: "a@b.com"})
	store.CreateUser(&core.User{ID: "u2", Email: "c@d.com"})

	if store.Count() != 2 {
		t.Fatalf("expected count 2, got %d", store.Count())
	}

	store.Clear()

	if store.Count() != 0 {
		t.Fatalf("expected count 0 after clear, got %d", store.Count())
	}
}
