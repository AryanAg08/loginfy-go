# Authentication Strategies

Strategies are the core abstraction in Lognify.go for handling authentication. Each strategy encapsulates a specific authentication method (email/password, OAuth, API keys, etc.).

## How Strategies Work

Every strategy implements the `Strategy` interface:

```go
type Strategy interface {
    Name() string
    Authenticate(ctx *Context) (*User, error)
}
```

- **`Name()`** returns a unique identifier for the strategy (e.g., `"email_password"` for the built-in email/password strategy)
- **`Authenticate()`** takes a `Context` containing request data and returns a `User` on success

Strategies are registered with Loginfy using `Use()`:

```go
app := core.New()
app.Use(emailPassword.New())
```

You can register multiple strategies and select one at authentication time:

```go
user, err := app.Authenticate("email_password", ctx)
```

## Email/Password Strategy

The built-in `emailPassword` strategy handles credential-based authentication.

### Setup

```go
import "github.com/AryanAg08/loginfy-go/strategies/emailPassword"

strategy := emailPassword.New()
app.Use(strategy)
```

### Registration

The strategy provides a `Register()` method (beyond the `Strategy` interface) for creating users:

```go
strategy, _ := app.GetStrategy("email_password")
ep := strategy.(*emailPassword.EmailPasswordStrategy)

ctx := &core.Context{Loginfy: app, RequestID: "reg-1"}
ctx.Set("email", "user@example.com")
ctx.Set("password", "mypassword")

user, err := ep.Register(ctx)
```

During registration:
1. Email and password are extracted from context
2. Password is hashed using bcrypt (via `pkg/crypto`)
3. A unique user ID is generated using `crypto/rand`
4. The user is stored via the configured storage adapter
5. The user object is returned (with hashed password)

### Authentication

```go
ctx := &core.Context{Loginfy: app, RequestID: "auth-1"}
ctx.Set("email", "user@example.com")
ctx.Set("password", "mypassword")

user, err := app.Authenticate("email_password", ctx)
```

During authentication:
1. Email and password are extracted from context
2. User is fetched from storage by email
3. Password is verified against the bcrypt hash
4. On success, the user is stored in context and `OnLogin` hook fires

### Error Handling

```go
import "github.com/AryanAg08/loginfy-go/strategies/emailPassword"

switch err {
case emailPassword.ErrMissingCredentials:
    // Email or password was empty
case emailPassword.ErrInvalidCredentials:
    // User not found or password mismatch
case core.ErrStorageNotSet:
    // No storage adapter configured
}
```

## Creating a Custom Strategy

You can implement the `Strategy` interface to create any authentication method:

```go
package apikey

import "github.com/AryanAg08/loginfy-go/core"

type APIKeyStrategy struct {
    // your fields
}

func New() *APIKeyStrategy {
    return &APIKeyStrategy{}
}

func (s *APIKeyStrategy) Name() string {
    return "api-key"
}

func (s *APIKeyStrategy) Authenticate(ctx *core.Context) (*core.User, error) {
    key := ctx.GetString("api_key")
    if key == "" {
        return nil, errors.New("API key required")
    }

    // Look up the user by API key
    storage := ctx.Loginfy.GetStorage()
    // ... your lookup logic ...

    return user, nil
}
```

Register it like any other strategy:

```go
app.Use(apikey.New())

// Use it
ctx.Set("api_key", "sk_live_abc123")
user, err := app.Authenticate("api-key", ctx)
```

### Guidelines for Custom Strategies

1. **Use `ctx.GetString()`** to read input data from the context
2. **Access storage** via `ctx.Loginfy.GetStorage()`
3. **Return sentinel errors** for known failure modes
4. **Log operations** using the `pkg/logger` package
5. **Never store plaintext passwords** — use `pkg/crypto` for hashing
6. **Strategy names must be unique** — duplicate names will overwrite

## Strategy Context Data Flow

The `Context` object is the bridge between your application and strategies:

```
Application                    Strategy
    │                             │
    ├── ctx.Set("email", "...")   │
    ├── ctx.Set("password", "...") │
    │                             │
    ├── Authenticate(ctx) ────────►│
    │                             ├── ctx.GetString("email")
    │                             ├── ctx.GetString("password")
    │                             ├── storage.GetUserByEmail(...)
    │                             ├── crypto.VerifyPassword(...)
    │◄──── (*User, nil) ──────────┤
    │                             │
    ├── ctx.GetUser() // user is set
```

## Next Steps

- [Getting Started](getting-started.md) — Quick start guide
- [Authorization](authorization.md) — RBAC and policy-based access control
- [Core API Reference](../api/core.md) — Full API documentation
