package tests_test

import (
	"testing"

	"github.com/AryanAg08/loginfy-go/authorization"
	"github.com/AryanAg08/loginfy-go/core"
)

func TestDefineRole(t *testing.T) {
	auth := authorization.New()
	auth.DefineRole("admin", "read", "write", "delete")

	perms := auth.GetRolePermissions("admin")
	if len(perms) != 3 {
		t.Fatalf("expected 3 permissions, got %d", len(perms))
	}

	// Check undefined role
	perms = auth.GetRolePermissions("nonexistent")
	if perms != nil {
		t.Fatalf("expected nil for undefined role, got %v", perms)
	}
}

func TestHasPermission(t *testing.T) {
	auth := authorization.New()
	auth.DefineRole("admin", "read", "write", "delete")
	auth.DefineRole("viewer", "read")

	admin := &core.User{ID: "u1", Roles: []string{"admin"}}
	viewer := &core.User{ID: "u2", Roles: []string{"viewer"}}

	if !auth.HasPermission(admin, "delete") {
		t.Fatal("expected admin to have 'delete' permission")
	}
	if auth.HasPermission(viewer, "delete") {
		t.Fatal("expected viewer NOT to have 'delete' permission")
	}
	if !auth.HasPermission(viewer, "read") {
		t.Fatal("expected viewer to have 'read' permission")
	}

	// Nil user
	if auth.HasPermission(nil, "read") {
		t.Fatal("expected nil user to not have any permission")
	}

	// User with nil roles
	noRoles := &core.User{ID: "u3"}
	if auth.HasPermission(noRoles, "read") {
		t.Fatal("expected user with nil roles to not have any permission")
	}
}

func TestGrantRevoke(t *testing.T) {
	auth := authorization.New()
	auth.DefineRole("editor", "read")

	user := &core.User{ID: "u1", Roles: []string{"editor"}}

	if auth.HasPermission(user, "write") {
		t.Fatal("expected editor NOT to have 'write' permission initially")
	}

	auth.GrantPermission("editor", "write")
	if !auth.HasPermission(user, "write") {
		t.Fatal("expected editor to have 'write' permission after grant")
	}

	auth.RevokePermission("editor", "write")
	if auth.HasPermission(user, "write") {
		t.Fatal("expected editor NOT to have 'write' permission after revoke")
	}
}

func TestAllowPolicy(t *testing.T) {
	auth := authorization.New()

	type Document struct {
		OwnerID string
	}

	auth.AllowPolicy("edit_document", func(user *core.User, resource interface{}) bool {
		doc, ok := resource.(*Document)
		if !ok {
			return false
		}
		return doc.OwnerID == user.ID
	})

	owner := &core.User{ID: "u1"}
	other := &core.User{ID: "u2"}
	doc := &Document{OwnerID: "u1"}

	if !auth.Can(owner, "edit_document", doc) {
		t.Fatal("expected owner to be able to edit document")
	}
	if auth.Can(other, "edit_document", doc) {
		t.Fatal("expected non-owner to NOT be able to edit document")
	}
}

func TestCan(t *testing.T) {
	auth := authorization.New()
	user := &core.User{ID: "u1"}

	// Policy not defined
	if auth.Can(user, "unknown_action", nil) {
		t.Fatal("expected Can to return false for undefined policy")
	}

	// Register a simple policy
	auth.AllowPolicy("view_dashboard", func(u *core.User, resource interface{}) bool {
		return u.ID != ""
	})

	if !auth.Can(user, "view_dashboard", nil) {
		t.Fatal("expected Can to return true for authorized user")
	}
}
