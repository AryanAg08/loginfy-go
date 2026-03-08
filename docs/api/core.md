# Core API Reference

Package: `github.com/AryanAg08/loginfy-go/core`

The core package defines the central types and interfaces for the Lognify.go framework.

## Loginfy

The main authentication framework instance.

```go
type Loginfy struct {
    // unexported fields
}
```

### `core.New() *Loginfy`

Creates a new Loginfy instance with an empty strategy registry.

```go
app := core.New()
```

### `(*Loginfy) Use(strategy Strategy)`

Registers an authentication strategy. The strategy is stored by its `Name()`.

```go
app.Use(emailPassword.New())
```

### `(*Loginfy) SetStorage(storage Storage)`

Sets the storage adapter for user persistence.

```go
app.SetStorage(memory.New())
```

### `(*Loginfy) GetStorage() Storage`

Returns the configured storage adapter, or `nil` if not set.

### `(*Loginfy) SetSessionManager(session SessionManager)`

Sets the session manager for token-based sessions.

```go
app.SetSessionManager(jwt.New(jwt.Config{Secret: "..."}))
```

### `(*Loginfy) GetSessionManager() SessionManager`

Returns the configured session manager, or `nil` if not set.

### `(*Loginfy) SetHooks(hooks Hooks)`

Sets lifecycle hooks for authentication events.

```go
app.SetHooks(core.Hooks{
    OnLogin:    func(user *core.User) { /* ... */ },
    OnRegister: func(user *core.User) { /* ... */ },
})
```

### `(*Loginfy) GetStrategy(name string) (Strategy, bool)`

Retrieves a registered strategy by name. Returns the strategy and `true` if found, or `nil` and `false` otherwise.

```go
strategy, ok := app.GetStrategy("email_password")
```

### `(*Loginfy) Authenticate(strategyName string, ctx *Context) (*User, error)`

Authenticates a user using the named strategy. On success, stores the user in context and fires the `OnLogin` hook.

**Errors:** `ErrStrategyNotFound`, strategy-specific errors.

```go
user, err := app.Authenticate("email_password", ctx)
```

### `(*Loginfy) Mount() func(http.Handler) http.Handler`

Returns HTTP middleware that creates a Loginfy `Context` for each request and stores it in the request's `context.Context`. Required for `RequireAuthWithLoginfy`, `RequireRole`, and `RequirePermission` middleware.

```go
handler := app.Mount()(mux)
```

### `(*Loginfy) Login(user *User) (string, error)`

Creates a session for the user via the session manager. Returns a token string.

**Errors:** `ErrSessionManagerNotSet`, session manager errors.

```go
token, err := app.Login(user)
```

### `(*Loginfy) Logout(ctx *Context, token string) error`

Destroys a user's session.

**Errors:** `ErrSessionManagerNotSet`, session manager errors.

```go
err := app.Logout(ctx, token)
```

---

## User

Represents an authenticated user.

```go
type User struct {
    ID        string                 `json:"id"`
    Email     string                 `json:"email"`
    Password  string                 `json:"-"`
    Roles     []string               `json:"roles"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
    CreatedAt time.Time              `json:"created_at"`
    UpdatedAt time.Time              `json:"updated_at"`
}
```

> **Note:** The `Password` field is tagged `json:"-"` and is never serialized in JSON output.

### `(*User) HasRole(role string) bool`

Returns `true` if the user has the specified role.

### `(*User) HasAnyRole(roles ...string) bool`

Returns `true` if the user has at least one of the specified roles.

---

## Context

Request-scoped context for authentication operations.

```go
type Context struct {
    Request   *http.Request
    Response  http.ResponseWriter
    Loginfy   *Loginfy
    RequestID string
}
```

### `(*Context) Set(key string, value interface{})`

Stores a key-value pair in the context.

### `(*Context) Get(key string) (interface{}, bool)`

Retrieves a value by key. Returns the value and `true` if found.

### `(*Context) GetString(key string) string`

Retrieves a string value by key. Returns empty string if not found or not a string.

### `(*Context) SetUser(user *User)`

Stores the authenticated user in the context.

### `(*Context) GetUser() (*User, bool)`

Retrieves the authenticated user. Returns the user and `true` if set.

### `(*Context) HasUser() bool`

Returns `true` if a user has been set in the context.

### `core.ContextWithLoginfy(ctx context.Context, loginfyCtx *Context) context.Context`

Stores a Loginfy `Context` in a standard `context.Context`. Used by `Mount()` middleware.

### `core.LoginfyFromContext(ctx context.Context) (*Context, bool)`

Retrieves the Loginfy `Context` from a standard `context.Context`.

```go
loginfyCtx, ok := core.LoginfyFromContext(r.Context())
```

---

## Strategy (Interface)

```go
type Strategy interface {
    Name() string
    Authenticate(ctx *Context) (*User, error)
}
```

| Method | Description |
|--------|-------------|
| `Name()` | Returns the unique identifier for this strategy |
| `Authenticate(ctx)` | Authenticates a user using data from the context |

---

## Storage (Interface)

```go
type Storage interface {
    CreateUser(user *User) error
    GetUserByEmail(email string) (*User, error)
    GetUserById(id string) (*User, error)
    UpdateUser(user *User) error
    DeleteUser(id string) error
}
```

| Method | Description |
|--------|-------------|
| `CreateUser(user)` | Persists a new user |
| `GetUserByEmail(email)` | Retrieves a user by email address |
| `GetUserById(id)` | Retrieves a user by ID |
| `UpdateUser(user)` | Updates an existing user |
| `DeleteUser(id)` | Removes a user by ID |

All implementations must be safe for concurrent use.

---

## SessionManager (Interface)

```go
type SessionManager interface {
    CreateSession(userId string) (string, error)
    ValidateSession(ctx *Context, token string) (string, error)
    DestroySession(ctx *Context, token string) error
}
```

| Method | Description |
|--------|-------------|
| `CreateSession(userId)` | Creates a new session, returns a token |
| `ValidateSession(ctx, token)` | Validates a token, returns the user ID |
| `DestroySession(ctx, token)` | Invalidates a session |

---

## Hooks

```go
type Hooks struct {
    OnLogin    func(user *User)
    OnRegister func(user *User)
}
```

| Field | Trigger |
|-------|---------|
| `OnLogin` | Called after successful `Authenticate()` |
| `OnRegister` | Called after successful user registration |

---

## Errors

| Error | Description |
|-------|-------------|
| `ErrStrategyNotFound` | Named strategy is not registered |
| `ErrSessionManagerNotSet` | Session manager has not been configured |
| `ErrStorageNotSet` | Storage adapter has not been configured |
| `ErrUnauthorized` | Authentication failed |
| `ErrInsufficientRole` | User lacks required role |
| `ErrInsufficientPermission` | User lacks required permission |
