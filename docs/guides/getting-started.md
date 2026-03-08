# Getting Started with Lognify.go

This guide walks you through setting up authentication in a Go application using Lognify.go.

## Installation

```bash
go get github.com/AryanAg08/loginfy.go
```

## Basic Setup

Every Lognify application follows the same pattern:

1. Create a `Loginfy` instance
2. Register an authentication strategy
3. Set a storage adapter
4. Set a session manager

```go
package main

import (
    "github.com/AryanAg08/loginfy.go/core"
    "github.com/AryanAg08/loginfy.go/strategies/emailPassword"
    "github.com/AryanAg08/loginfy.go/storage/memory"
    "github.com/AryanAg08/loginfy.go/sessions/jwt"
)

func main() {
    // 1. Create instance
    app := core.New()

    // 2. Register strategy
    app.Use(emailPassword.New())

    // 3. Set storage
    app.SetStorage(memory.New())

    // 4. Set session manager
    app.SetSessionManager(jwt.New(jwt.Config{
        Secret: "your-secret-key-at-least-32-chars!!",
    }))
}
```

## Register a User

Use the email/password strategy's `Register` method to create users:

```go
strategy, _ := app.GetStrategy("email_password")
ep := strategy.(*emailPassword.EmailPasswordStrategy)

ctx := &core.Context{Loginfy: app, RequestID: "register-1"}
ctx.Set("email", "alice@example.com")
ctx.Set("password", "strongPassword123")

user, err := ep.Register(ctx)
if err != nil {
    log.Fatalf("Registration failed: %v", err)
}
fmt.Printf("User registered: %s\n", user.ID)
```

The strategy automatically:
- Hashes the password with bcrypt
- Generates a unique user ID
- Stores the user via the configured storage adapter

## Authenticate

```go
ctx := &core.Context{Loginfy: app, RequestID: "login-1"}
ctx.Set("email", "alice@example.com")
ctx.Set("password", "strongPassword123")

user, err := app.Authenticate("email_password", ctx)
if err != nil {
    log.Fatalf("Authentication failed: %v", err)
}

// Create a JWT session token
token, err := app.Login(user)
if err != nil {
    log.Fatalf("Session creation failed: %v", err)
}
fmt.Printf("JWT Token: %s\n", token)
```

## Protect Routes

Use middleware to protect your HTTP endpoints:

```go
import (
    "net/http"
    "github.com/AryanAg08/loginfy.go/middleware"
)

mux := http.NewServeMux()

// Public route
mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte(`{"status":"ok"}`))
})

// Protected route — requires valid Authorization header
mux.Handle("/api/data", middleware.RequireAuth(
    http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte(`{"data":"secret"}`))
    }),
))

// Protected route — validates JWT and loads user
mux.Handle("/api/profile", middleware.RequireAuthWithLoginfy(app)(
    http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx, _ := core.LoginfyFromContext(r.Context())
        user, _ := ctx.GetUser()
        fmt.Fprintf(w, `{"email":"%s"}`, user.Email)
    }),
))

// Wrap with Loginfy context middleware
handler := app.Mount()(mux)
http.ListenAndServe(":8080", handler)
```

## Full Working Example

```go
package main

import (
    "fmt"
    "log"
    "net/http"
    "time"

    "github.com/AryanAg08/loginfy.go/core"
    "github.com/AryanAg08/loginfy.go/middleware"
    "github.com/AryanAg08/loginfy.go/sessions/jwt"
    "github.com/AryanAg08/loginfy.go/storage/memory"
    "github.com/AryanAg08/loginfy.go/strategies/emailPassword"
)

func main() {
    // Setup
    app := core.New()
    app.Use(emailPassword.New())
    app.SetStorage(memory.New())
    app.SetSessionManager(jwt.New(jwt.Config{
        Secret:     "my-super-secret-key-for-jwt-sign!",
        Expiration: 1 * time.Hour,
    }))

    // Set hooks
    app.SetHooks(core.Hooks{
        OnLogin: func(user *core.User) {
            fmt.Printf("[Hook] User logged in: %s\n", user.Email)
        },
    })

    // Register a test user
    strategy, _ := app.GetStrategy("email_password")
    ep := strategy.(*emailPassword.EmailPasswordStrategy)

    regCtx := &core.Context{Loginfy: app, RequestID: "setup"}
    regCtx.Set("email", "demo@example.com")
    regCtx.Set("password", "demo1234")
    ep.Register(regCtx)

    // Routes
    mux := http.NewServeMux()

    mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
        ctx := &core.Context{
            Request:   r,
            Response:  w,
            Loginfy:   app,
            RequestID: fmt.Sprintf("req-%d", time.Now().UnixNano()),
        }
        ctx.Set("email", r.URL.Query().Get("email"))
        ctx.Set("password", r.URL.Query().Get("password"))

        user, err := app.Authenticate("email_password", ctx)
        if err != nil {
            http.Error(w, err.Error(), http.StatusUnauthorized)
            return
        }

        token, _ := app.Login(user)
        fmt.Fprintf(w, `{"token":"%s"}`, token)
    })

    mux.Handle("/protected", middleware.RequireAuth(
        http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            fmt.Fprintln(w, `{"message":"welcome!"}`)
        }),
    ))

    handler := app.Mount()(mux)
    log.Println("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", handler))
}
```

## Next Steps

- [Strategies Guide](strategies.md) — Learn how authentication strategies work
- [Authorization Guide](authorization.md) — Set up RBAC and policy-based access control
- [Storage Adapters Guide](storage-adapters.md) — Use or build storage backends
- [API Reference](../api/core.md) — Full API documentation
