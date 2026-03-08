# Authorization Guide

Lognify.go provides two authorization models: **Role-Based Access Control (RBAC)** and **Policy-Based Authorization**. Both are managed through the `Authorizer` in the `authorization` package.

## RBAC Setup

### Defining Roles

Roles are defined with a name and a set of permissions:

```go
import "github.com/AryanAg08/loginfy-go/authorization"

auth := authorization.New()

// Define roles with permissions
auth.DefineRole("admin", "users:read", "users:write", "users:delete", "posts:read", "posts:write", "posts:delete")
auth.DefineRole("editor", "posts:read", "posts:write")
auth.DefineRole("viewer", "posts:read")
```

### Managing Permissions

You can add or remove individual permissions after defining a role:

```go
// Grant a new permission to an existing role
auth.GrantPermission("editor", "posts:publish")

// Revoke a permission from a role
auth.RevokePermission("editor", "posts:publish")

// Inspect a role's permissions
perms := auth.GetRolePermissions("editor")
// ["posts:read", "posts:write"]
```

### Assigning Roles to Users

Roles are stored on the `User` object:

```go
user := &core.User{
    ID:    "user-123",
    Email: "editor@example.com",
    Roles: []string{"editor"},
}

// Check roles on the user directly
user.HasRole("editor")          // true
user.HasAnyRole("admin", "editor") // true
```

### Checking Permissions

The `Authorizer` checks if **any** of the user's roles grant a permission:

```go
if auth.HasPermission(user, "posts:write") {
    // User has "posts:write" via their "editor" role
    fmt.Println("User can write posts")
}

if !auth.HasPermission(user, "users:delete") {
    // Editor role doesn't include "users:delete"
    fmt.Println("User cannot delete users")
}
```

## Policy-Based Authorization

Policies provide fine-grained, resource-level access control. A policy is a function that decides if a user can perform an action on a specific resource.

### Defining Policies

```go
type Post struct {
    ID       string
    AuthorID string
    Status   string
}

// Only the author or an admin can edit a post
auth.AllowPolicy("edit-post", func(user *core.User, resource interface{}) bool {
    post := resource.(*Post)
    return post.AuthorID == user.ID || user.HasRole("admin")
})

// Only admins can delete published posts
auth.AllowPolicy("delete-post", func(user *core.User, resource interface{}) bool {
    post := resource.(*Post)
    if post.Status == "published" {
        return user.HasRole("admin")
    }
    return post.AuthorID == user.ID
})
```

### Evaluating Policies

```go
post := &Post{ID: "post-1", AuthorID: "user-123", Status: "draft"}

if auth.Can(user, "edit-post", post) {
    fmt.Println("User can edit this post")
}

if auth.Can(user, "delete-post", post) {
    fmt.Println("User can delete this post")
}
```

If a policy for the given action doesn't exist, `Can()` returns `false`.

## Middleware Integration

### RequireRole Middleware

Protect routes by requiring the authenticated user to have specific roles:

```go
import "github.com/AryanAg08/loginfy-go/middleware"

// Require "admin" role
mux.Handle("/admin/dashboard",
    middleware.RequireAuthWithLoginfy(app)(
        middleware.RequireRole(app, "admin")(
            http.HandlerFunc(adminDashboard),
        ),
    ),
)

// Require "admin" OR "editor" role
mux.Handle("/posts/edit",
    middleware.RequireAuthWithLoginfy(app)(
        middleware.RequireRole(app, "admin", "editor")(
            http.HandlerFunc(editPost),
        ),
    ),
)
```

### RequirePermission Middleware

Check for specific permissions stored in user metadata:

```go
// User must have "posts:delete" in their metadata permissions
mux.Handle("/posts/delete",
    middleware.RequireAuthWithLoginfy(app)(
        middleware.RequirePermission(app, "posts:delete")(
            http.HandlerFunc(deletePost),
        ),
    ),
)
```

> **Note:** `RequirePermission` checks `user.Metadata["permissions"]` (a `[]string`), while `Authorizer.HasPermission()` checks role-based permission mappings. Use them for different authorization models or combine them.

### Middleware Chain Example

A complete middleware chain for a protected admin endpoint:

```go
app := core.New()
app.Use(emailPassword.New())
app.SetStorage(memory.New())
app.SetSessionManager(jwt.New(jwt.Config{Secret: "secret"}))

mux := http.NewServeMux()

// Public
mux.HandleFunc("/health", healthHandler)

// Authenticated
mux.Handle("/api/profile",
    middleware.RequireAuthWithLoginfy(app)(
        http.HandlerFunc(profileHandler),
    ),
)

// Admin only
mux.Handle("/api/admin",
    middleware.RequireAuthWithLoginfy(app)(
        middleware.RequireRole(app, "admin")(
            http.HandlerFunc(adminHandler),
        ),
    ),
)

// Mount Loginfy context and start server
handler := app.Mount()(mux)
http.ListenAndServe(":8080", handler)
```

## Combining RBAC and Policies

You can use both RBAC and policies together for layered authorization:

```go
auth := authorization.New()

// RBAC for broad access
auth.DefineRole("editor", "posts:read", "posts:write")

// Policy for resource-specific checks
auth.AllowPolicy("edit-post", func(user *core.User, resource interface{}) bool {
    post := resource.(*Post)
    // Must have the permission AND be the author (or admin)
    return auth.HasPermission(user, "posts:write") &&
        (post.AuthorID == user.ID || user.HasRole("admin"))
})
```

## Next Steps

- [Strategies Guide](strategies.md) — Authentication strategies
- [Storage Adapters](storage-adapters.md) — Storage backends
- [Middleware API Reference](../api/middleware.md) — Full middleware docs
- [Authorization API Reference](../api/authorization.md) — Full authorization docs
