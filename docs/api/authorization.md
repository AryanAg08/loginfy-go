# Authorization API Reference

Package: `github.com/AryanAg08/loginfy-go/authorization`

The authorization package provides role-based access control (RBAC) and policy-based authorization.

## Types

### PolicyFunc

```go
type PolicyFunc func(user *core.User, resource interface{}) bool
```

A function that determines whether a user can perform an action on a resource. Returns `true` to allow, `false` to deny.

### Authorizer

```go
type Authorizer struct {
    // unexported fields
}
```

Manages role-permission mappings and policy functions.

---

## Functions

### `authorization.New() *Authorizer`

Creates a new `Authorizer` with empty role and policy registries.

```go
auth := authorization.New()
```

---

## Methods

### `(*Authorizer) DefineRole(role string, permissions ...string)`

Defines a role with the given permissions. If the role already exists, permissions are added to it.

```go
auth.DefineRole("admin", "users:read", "users:write", "users:delete")
auth.DefineRole("editor", "posts:read", "posts:write")
```

### `(*Authorizer) GrantPermission(role, permission string)`

Adds a single permission to a role. Creates the role if it doesn't exist.

```go
auth.GrantPermission("editor", "posts:publish")
```

### `(*Authorizer) RevokePermission(role, permission string)`

Removes a single permission from a role. No-op if the role or permission doesn't exist.

```go
auth.RevokePermission("editor", "posts:publish")
```

### `(*Authorizer) HasPermission(user *core.User, permission string) bool`

Returns `true` if **any** of the user's roles grant the specified permission.

Returns `false` if:
- `user` is `nil`
- `user.Roles` is `nil`
- None of the user's roles have the permission

```go
user := &core.User{Roles: []string{"editor"}}

auth.HasPermission(user, "posts:write")  // true
auth.HasPermission(user, "users:delete") // false
```

### `(*Authorizer) AllowPolicy(action string, fn PolicyFunc)`

Registers a policy function for an action. If a policy already exists for the action, it is overwritten.

```go
auth.AllowPolicy("edit-post", func(user *core.User, resource interface{}) bool {
    post := resource.(*Post)
    return post.AuthorID == user.ID || user.HasRole("admin")
})
```

### `(*Authorizer) Can(user *core.User, action string, resource interface{}) bool`

Evaluates the policy registered for the given action. Returns `false` if no policy is registered for the action.

```go
if auth.Can(user, "edit-post", post) {
    // Allowed
}
```

### `(*Authorizer) GetRolePermissions(role string) []string`

Returns a slice of all permissions granted to the specified role. Returns `nil` if the role doesn't exist.

```go
perms := auth.GetRolePermissions("admin")
// ["users:read", "users:write", "users:delete"]
```

> **Note:** The order of returned permissions is not guaranteed.

---

## Usage Patterns

### RBAC Only

```go
auth := authorization.New()
auth.DefineRole("admin", "users:manage", "posts:manage")
auth.DefineRole("user", "posts:read", "posts:create")

if auth.HasPermission(currentUser, "users:manage") {
    // Admin-level action
}
```

### Policy Only

```go
auth := authorization.New()

auth.AllowPolicy("view-profile", func(user *core.User, resource interface{}) bool {
    profile := resource.(*UserProfile)
    return profile.IsPublic || profile.UserID == user.ID
})

if auth.Can(currentUser, "view-profile", targetProfile) {
    // Show profile
}
```

### RBAC + Policies Combined

```go
auth := authorization.New()
auth.DefineRole("editor", "posts:write")

auth.AllowPolicy("edit-post", func(user *core.User, resource interface{}) bool {
    post := resource.(*Post)
    // Must have permission AND be the author (or admin)
    return auth.HasPermission(user, "posts:write") &&
        (post.AuthorID == user.ID || user.HasRole("admin"))
})
```
