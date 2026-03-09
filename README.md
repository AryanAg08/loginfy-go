<div align="center">

# 🔐 Loginfy-go

### Plug-and-play authentication and authorization framework for Go applications

[![Go Reference](https://pkg.go.dev/badge/github.com/AryanAg08/loginfy-go.svg)](https://pkg.go.dev/github.com/AryanAg08/loginfy-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-passing-brightgreen.svg)]()

</div>

---

**Lognify.go** (module: `loginfy.go`) is a modular, extensible authentication and authorization framework for Go. It provides everything you need to add secure auth to your application — strategies, sessions, RBAC, policy-based authorization, middleware, hooks, and structured logging — all with a clean, composable API.

## ✨ Features

- **Modular Auth Strategies** — Plug in email/password, OAuth (coming soon), or build your own
- **JWT Session Management** — Stateless token-based sessions with HMAC-SHA256 signing
- **Role-Based Access Control (RBAC)** — Define roles, assign permissions, enforce with middleware
- **Policy-Based Authorization** — Fine-grained resource-level access control
- **Storage Adapters** — In-memory (built-in), MongoDB (planned), or implement your own
- **HTTP Middleware** — `RequireAuth`, `RequireRole`, `RequirePermission` for `net/http`
- **Lifecycle Hooks** — `OnLogin`, `OnRegister` callbacks for custom logic
- **Structured Logging** — Built-in service logger with sessions, levels, and colored output

## 🚀 Quick Start

```go
package main

import (
    "fmt"
    "github.com/AryanAg08/loginfy-go/core"
    "github.com/AryanAg08/loginfy-go/strategies/emailPassword"
    "github.com/AryanAg08/loginfy-go/storage/memory"
    "github.com/AryanAg08/loginfy-go/sessions/jwt"
)

func main() {
    app := core.New()
    app.Use(emailPassword.New())
    app.SetStorage(memory.New())
    app.SetSessionManager(jwt.New(jwt.Config{Secret: "my-secret"}))

    // Register, authenticate, create sessions — you're ready!
    fmt.Println("Lognify is running!")
}
```

## 📦 Installation

```bash
go get github.com/AryanAg08/loginfy-go
```

Requires **Go 1.21+**.

## 📖 Usage

### Basic Setup with Email/Password

```go
package main

import (
    "fmt"
    "time"

    "github.com/AryanAg08/loginfy-go/core"
    "github.com/AryanAg08/loginfy-go/strategies/emailPassword"
    "github.com/AryanAg08/loginfy-go/storage/memory"
    "github.com/AryanAg08/loginfy-go/sessions/jwt"
)

func main() {
    // 1. Create the Loginfy instance
    app := core.New()

    // 2. Register the email/password strategy
    app.Use(emailPassword.New())

    // 3. Set up storage and session manager
    app.SetStorage(memory.New())
    app.SetSessionManager(jwt.New(jwt.Config{
        Secret:     "your-secret-key",
        Expiration: 24 * time.Hour,
    }))

    // 4. Register a user
    strategy, _ := app.GetStrategy("email_password")
    ep := strategy.(*emailPassword.EmailPasswordStrategy)

    ctx := &core.Context{Loginfy: app, RequestID: "setup"}
    ctx.Set("email", "user@example.com")
    ctx.Set("password", "securepass123")

    user, err := ep.Register(ctx)
    if err != nil {
        panic(err)
    }
    fmt.Printf("Registered: %s (%s)\n", user.Email, user.ID)

    // 5. Authenticate
    authCtx := &core.Context{Loginfy: app, RequestID: "login"}
    authCtx.Set("email", "user@example.com")
    authCtx.Set("password", "securepass123")

    user, err = app.Authenticate("email_password", authCtx)
    if err != nil {
        panic(err)
    }

    // 6. Create a JWT session
    token, _ := app.Login(user)
    fmt.Printf("Token: %s\n", token)
}
```

### OAuth (Planned for v0.2)

OAuth strategies (Google, GitHub, etc.) are on the roadmap. You'll be able to add them the same way:

```go
// Coming in v0.2
app.Use(oauth.NewGoogle(oauth.GoogleConfig{
    ClientID:     "...",
    ClientSecret: "...",
    RedirectURL:  "http://localhost:8080/callback",
}))
```

### JWT Session Management

```go
import "github.com/AryanAg08/loginfy-go/sessions/jwt"

sm := jwt.New(jwt.Config{
    Secret:     "your-256-bit-secret",
    Expiration: 2 * time.Hour,
})

// Create a session token
token, err := sm.CreateSession(user.ID)

// Create with full user details embedded
token, err = sm.CreateSessionWithUser(user)

// Validate a token
userID, err := sm.ValidateSession(ctx, token)

// Validate and get full claims
claims, err := sm.ValidateSessionWithClaims(ctx, token)
// claims.UserID, claims.Email, claims.Roles, claims.ExpiresAt

// Destroy session (logout)
err = sm.DestroySession(ctx, token)
```

### RBAC Authorization

```go
import "github.com/AryanAg08/loginfy-go/authorization"

auth := authorization.New()

// Define roles with permissions
auth.DefineRole("admin", "users:read", "users:write", "users:delete")
auth.DefineRole("editor", "posts:read", "posts:write")
auth.DefineRole("viewer", "posts:read")

// Grant/revoke individual permissions
auth.GrantPermission("editor", "posts:delete")
auth.RevokePermission("editor", "posts:delete")

// Check permissions
if auth.HasPermission(user, "users:write") {
    // User has permission via one of their roles
}
```

### Policy-Based Authorization

```go
auth := authorization.New()

// Define policies for fine-grained access control
auth.AllowPolicy("edit-post", func(user *core.User, resource interface{}) bool {
    post := resource.(*Post)
    return post.AuthorID == user.ID || user.HasRole("admin")
})

// Check policy
if auth.Can(user, "edit-post", post) {
    // User can edit this specific post
}
```

### Middleware Usage

```go
import "github.com/AryanAg08/loginfy-go/middleware"

mux := http.NewServeMux()

// Mount Loginfy context (required for other middleware)
handler := app.Mount()(mux)

// Require any valid auth token
mux.Handle("/api/data", middleware.RequireAuth(dataHandler))

// Require valid JWT + load user into context
mux.Handle("/api/profile",
    middleware.RequireAuthWithLoginfy(app)(profileHandler))

// Require specific roles
mux.Handle("/api/admin",
    middleware.RequireAuthWithLoginfy(app)(
        middleware.RequireRole(app, "admin")(adminHandler)))

// Require specific permission
mux.Handle("/api/posts/delete",
    middleware.RequireAuthWithLoginfy(app)(
        middleware.RequirePermission(app, "posts:delete")(deleteHandler)))
```

### Hooks

```go
app.SetHooks(core.Hooks{
    OnLogin: func(user *core.User) {
        fmt.Printf("User logged in: %s\n", user.Email)
        // Send notification, update last login, etc.
    },
    OnRegister: func(user *core.User) {
        fmt.Printf("New user registered: %s\n", user.Email)
        // Send welcome email, initialize defaults, etc.
    },
})
```

### Storage Adapters

```go
import "github.com/AryanAg08/loginfy-go/storage/memory"

// In-memory storage (great for development/testing)
store := memory.New()
app.SetStorage(store)

// Storage interface — implement for any backend:
// CreateUser, GetUserByEmail, GetUserById, UpdateUser, DeleteUser
```

**MongoDB** support is planned for a future release. See [Storage Adapters Guide](docs/guides/storage-adapters.md) for how to build your own.

## 🏗 Architecture

```
┌─────────────────────────────────────────────────────┐
│                   Your Application                  │
├─────────────────────────────────────────────────────┤
│                  HTTP Middleware                     │
│   RequireAuth │ RequireRole │ RequirePermission      │
├───────────────┬─────────────┬───────────────────────┤
│   Strategies  │  Sessions   │   Authorization       │
│  ┌──────────┐ │ ┌─────────┐ │  ┌─────────────────┐  │
│  │Email/Pass│ │ │   JWT   │ │  │  RBAC + Policy  │  │
│  │  OAuth*  │ │ │         │ │  │                 │  │
│  └──────────┘ │ └─────────┘ │  └─────────────────┘  │
├───────────────┴─────────────┴───────────────────────┤
│                    Core (Loginfy)                    │
│          Context │ User │ Hooks │ Errors            │
├─────────────────────────────────────────────────────┤
│                  Storage Adapters                   │
│              Memory │ MongoDB* │ Custom             │
├─────────────────────────────────────────────────────┤
│                   pkg/ Utilities                    │
│           crypto │ logger │ constants │ status      │
└─────────────────────────────────────────────────────┘
                    * = planned
```

## 📁 Project Structure

```
loginfy.go/
├── core/                   # Core types: Loginfy, User, Context, Strategy, Storage interfaces
│   ├── loginfy.go          # Main Loginfy struct and methods
│   ├── user.go             # User model with role helpers
│   ├── context.go          # Request context with data store
│   ├── startegy.go         # Strategy interface
│   ├── storage.go          # Storage interface
│   ├── session.go          # SessionManager interface
│   ├── hooks.go            # OnLogin/OnRegister hooks
│   └── errors.go           # Sentinel errors
├── strategies/
│   └── emailPassword/      # Email + password authentication strategy
├── sessions/
│   └── jwt/                # JWT session manager (HMAC-SHA256)
├── storage/
│   ├── memory/             # Thread-safe in-memory storage
│   └── mongodb/            # MongoDB adapter (placeholder)
├── authorization/          # RBAC roles/permissions + policy engine
├── middleware/              # HTTP middleware (RequireAuth, RequireRole, etc.)
├── pkg/
│   ├── crypto/             # Password hashing (bcrypt), token generation
│   ├── logger/             # Structured logging with sessions and service loggers
│   ├── constants/          # Shared constants
│   └── status/             # HTTP status helpers
├── examples/               # Working example applications
├── tests/                  # Test suite
└── docs/                   # Documentation
```

## ⚙️ Configuration

### JWT Session Config

| Field        | Type            | Default  | Description                 |
|-------------|-----------------|----------|-----------------------------|
| `Secret`    | `string`        | required | HMAC-SHA256 signing key     |
| `Expiration`| `time.Duration` | 24h      | Token expiration duration   |

### Logger Config

| Field        | Type     | Default      | Description                    |
|-------------|----------|--------------|--------------------------------|
| `Service`   | `string` | `""`         | Service name for log entries   |
| `Level`     | `Level`  | `INFO`       | Minimum log level              |
| `TimeFormat`| `string` | `RFC3339`    | Timestamp format               |
| `LogDir`    | `string` | `""`         | Directory for session log files|
| `UseColor`  | `bool`   | `false`      | Enable colored console output  |
| `JSONOutput`| `bool`   | `false`      | Output logs as JSON            |

## 🔒 Security Features

- **bcrypt Password Hashing** — Industry-standard adaptive hashing via `golang.org/x/crypto`
- **HMAC-SHA256 JWT Signing** — Tamper-proof stateless tokens
- **Constant-Time Comparison** — Prevents timing attacks on token validation
- **Cryptographically Secure Token Generation** — Uses `crypto/rand` for IDs and API keys
- **Password Never Serialized** — `User.Password` tagged with `json:"-"`
- **Structured Error Handling** — Sentinel errors prevent information leakage

## 🤝 Contributing

Contributions are welcome! Here's how to get started:

1. **Fork** the repository
2. **Clone** your fork: `git clone https://github.com/AryanAg08/loginfy-go.git`
3. **Create a branch**: `git checkout -b feature/my-feature`
4. **Make changes** and add tests
5. **Run tests**: `go test ./...`
6. **Commit**: `git commit -m "feat: add my feature"`
7. **Push**: `git push origin feature/my-feature`
8. **Open a Pull Request**

Please follow [Conventional Commits](https://www.conventionalcommits.org/) for commit messages.

## 📄 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

Copyright (c) 2026 Aryan Goyal

## 🗺 Roadmap

| Version | Milestone       | Features                                                        |
|---------|----------------|-----------------------------------------------------------------|
| v0.1 <b>(current)</b>   | Foundation     | Core framework, email/password, JWT, memory storage, middleware |
| v0.2    | OAuth          | Google, GitHub, Discord OAuth strategies                        |
| v0.3    | Authorization  | Enhanced RBAC, permission inheritance, audit logging            |
| v0.4    | Advanced       | MongoDB/PostgreSQL storage, rate limiting, 2FA, refresh tokens  |

---

<div align="center">
  Built with ❤️ by <a href="https://github.com/AryanAg08">Aryan Goyal</a>
</div>
