# Middleware API Reference

Package: `github.com/AryanAg08/loginfy-go/middleware`

HTTP middleware functions for `net/http` that handle authentication and authorization.

## RequireAuth

```go
func RequireAuth(next http.Handler) http.Handler
```

Checks that the `Authorization` header is present in the request. Does **not** validate the token — use this for simple token-presence checks.

**Behavior:**
- If `Authorization` header is empty → responds with `401 Unauthorized`
- If header is present → passes request to the next handler

**Usage:**

```go
mux.Handle("/api/data", middleware.RequireAuth(myHandler))
```

---

## RequireAuthWithLoginfy

```go
func RequireAuthWithLoginfy(loginfy *core.Loginfy) func(http.Handler) http.Handler
```

Validates the JWT token from the `Authorization` header and loads the authenticated user into the Loginfy context. This is the primary authentication middleware.

**Prerequisites:**
- `Loginfy.Mount()` middleware must be applied first (provides Loginfy context)
- A session manager must be configured on the Loginfy instance
- A storage adapter must be configured on the Loginfy instance

**Behavior:**
1. Extracts the Loginfy context from the request
2. Reads the `Authorization` header (strips `Bearer ` prefix if present)
3. Validates the token via `SessionManager.ValidateSession()`
4. Loads the user from `Storage.GetUserById()`
5. Stores the user in the Loginfy context via `ctx.SetUser()`
6. Passes request to the next handler

**Error Responses:**
- Missing Loginfy context → `500 Internal Server Error`
- Missing Authorization header → `401 Unauthorized`
- Missing session manager or storage → `500 Internal Server Error`
- Invalid/expired token → `401 Unauthorized`
- User not found → `401 Unauthorized`

**Usage:**

```go
handler := app.Mount()(mux)

mux.Handle("/api/profile",
    middleware.RequireAuthWithLoginfy(app)(
        http.HandlerFunc(profileHandler),
    ),
)
```

---

## RequireRole

```go
func RequireRole(loginfy *core.Loginfy, roles ...string) func(http.Handler) http.Handler
```

Ensures the authenticated user has **at least one** of the specified roles. Must be used after `RequireAuthWithLoginfy` so that a user is available in the context.

**Behavior:**
1. Extracts the Loginfy context from the request
2. Gets the authenticated user from context
3. Calls `user.HasAnyRole(roles...)` to check role membership
4. If the user has any matching role → passes to the next handler
5. Otherwise → responds with `403 Forbidden`

**Error Responses:**
- Missing Loginfy context → `500 Internal Server Error`
- No authenticated user → `403 Forbidden`
- User lacks required roles → `403 Forbidden`

**Usage:**

```go
// Single role
mux.Handle("/admin",
    middleware.RequireAuthWithLoginfy(app)(
        middleware.RequireRole(app, "admin")(adminHandler),
    ),
)

// Multiple roles (OR logic)
mux.Handle("/content",
    middleware.RequireAuthWithLoginfy(app)(
        middleware.RequireRole(app, "admin", "editor")(contentHandler),
    ),
)
```

---

## RequirePermission

```go
func RequirePermission(loginfy *core.Loginfy, permission string) func(http.Handler) http.Handler
```

Checks if the authenticated user has a specific permission in their metadata. Permissions are read from `user.Metadata["permissions"]` as a `[]string`.

**Behavior:**
1. Extracts the Loginfy context from the request
2. Gets the authenticated user from context
3. Reads `user.Metadata["permissions"]`
4. Checks if the required permission is in the list
5. If found → passes to the next handler
6. Otherwise → responds with `403 Forbidden`

**Error Responses:**
- Missing Loginfy context → `500 Internal Server Error`
- No authenticated user → `403 Forbidden`
- No metadata or no permissions key → `403 Forbidden`
- Permission not found → `403 Forbidden`

**Usage:**

```go
mux.Handle("/api/posts/delete",
    middleware.RequireAuthWithLoginfy(app)(
        middleware.RequirePermission(app, "posts:delete")(deleteHandler),
    ),
)
```

---

## Middleware Chaining

Middleware can be chained together for layered protection:

```go
// Public → Authenticated → Role-checked
mux.Handle("/admin/users",
    middleware.RequireAuthWithLoginfy(app)(
        middleware.RequireRole(app, "admin")(
            http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                ctx, _ := core.LoginfyFromContext(r.Context())
                user, _ := ctx.GetUser()
                fmt.Fprintf(w, "Welcome admin: %s", user.Email)
            }),
        ),
    ),
)
```

**Required order:**
1. `app.Mount()` — creates Loginfy context (outermost)
2. `RequireAuthWithLoginfy` — validates token, loads user
3. `RequireRole` or `RequirePermission` — checks access (innermost)
