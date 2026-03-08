package authorization

import (
	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

var log = logger.NewServiceLogger("authorization")

// PolicyFunc defines a function that checks whether a user can perform an action on a resource
type PolicyFunc func(user *core.User, resource interface{}) bool

// Authorizer manages roles, permissions, and policies
type Authorizer struct {
	// rolePermissions maps role names to their allowed permissions
	rolePermissions map[string]map[string]bool
	// policies maps action names to policy functions
	policies map[string]PolicyFunc
}

// New creates a new Authorizer
func New() *Authorizer {
	return &Authorizer{
		rolePermissions: make(map[string]map[string]bool),
		policies:        make(map[string]PolicyFunc),
	}
}

// DefineRole defines a role with a set of permissions
func (a *Authorizer) DefineRole(role string, permissions ...string) {
	if a.rolePermissions[role] == nil {
		a.rolePermissions[role] = make(map[string]bool)
	}
	for _, p := range permissions {
		a.rolePermissions[role][p] = true
	}
	log.Info("role defined", map[string]interface{}{
		"role":        role,
		"permissions": permissions,
	})
}

// GrantPermission adds a permission to an existing role
func (a *Authorizer) GrantPermission(role, permission string) {
	if a.rolePermissions[role] == nil {
		a.rolePermissions[role] = make(map[string]bool)
	}
	a.rolePermissions[role][permission] = true
}

// RevokePermission removes a permission from a role
func (a *Authorizer) RevokePermission(role, permission string) {
	if a.rolePermissions[role] != nil {
		delete(a.rolePermissions[role], permission)
	}
}

// HasPermission checks if any of the user's roles grants the specified permission
func (a *Authorizer) HasPermission(user *core.User, permission string) bool {
	if user == nil || user.Roles == nil {
		return false
	}
	for _, role := range user.Roles {
		if perms, ok := a.rolePermissions[role]; ok {
			if perms[permission] {
				return true
			}
		}
	}
	return false
}

// AllowPolicy registers a policy function for an action
func (a *Authorizer) AllowPolicy(action string, fn PolicyFunc) {
	a.policies[action] = fn
	log.Info("policy registered", map[string]interface{}{
		"action": action,
	})
}

// Can checks if a user can perform an action on a resource using policies
func (a *Authorizer) Can(user *core.User, action string, resource interface{}) bool {
	fn, ok := a.policies[action]
	if !ok {
		log.Warn("policy not found", map[string]interface{}{
			"action": action,
		})
		return false
	}
	return fn(user, resource)
}

// GetRolePermissions returns the permissions for a given role
func (a *Authorizer) GetRolePermissions(role string) []string {
	perms, ok := a.rolePermissions[role]
	if !ok {
		return nil
	}
	result := make([]string, 0, len(perms))
	for p := range perms {
		result = append(result, p)
	}
	return result
}
