# Lognify.go — Product Requirements Document (PRD)

## 1. Product Overview

**Lognify.go** (Go module: `github.com/AryanAg08/loginfy.go`) is a modular, plug-and-play authentication and authorization framework for Go applications. It provides a composable API for handling user registration, login, session management, role-based access control, policy-based authorization, and HTTP middleware integration.

The framework is designed with extensibility at its core — every major component (strategies, storage, sessions, authorization) is defined as an interface, allowing developers to swap implementations without changing application code.

## 2. Problem Statement

Go developers building web applications need authentication and authorization, but the ecosystem lacks a unified, modular framework that:

- Provides a clean abstraction over authentication strategies
- Supports multiple session backends (JWT, cookie, etc.)
- Integrates RBAC and policy-based authorization in one package
- Works with `net/http` out of the box
- Is easy to extend with custom strategies and storage adapters

Most teams end up building auth from scratch or gluing together disparate libraries, leading to inconsistent patterns, security gaps, and maintenance burden.

## 3. Goals

1. Provide a zero-config quick start for email/password authentication
2. Support pluggable authentication strategies (email/password now, OAuth next)
3. Offer JWT-based stateless session management
4. Include RBAC with role definitions and permission checks
5. Support policy-based authorization for resource-level access control
6. Provide HTTP middleware for `net/http` compatible routers
7. Include lifecycle hooks for login/register events
8. Ship with in-memory storage for development and a clear interface for production adapters
9. Provide structured logging throughout the framework
10. Maintain a clean, idiomatic Go API

## 4. Non-Goals

- Lognify.go is **not** a full web framework — it focuses solely on auth
- It does **not** provide a UI or frontend components
- It does **not** manage database migrations
- It does **not** handle email sending (verification, password reset) — hooks can be used to integrate external services

## 5. Target Users

- **Go backend developers** building REST APIs or web applications
- **Startup teams** who need production-ready auth without reinventing the wheel
- **Open-source maintainers** looking for a composable auth library
- **Students and learners** exploring authentication patterns in Go

## 6. Architecture

### 6.1 Core Layer (`core/`)

The core package defines the fundamental types and interfaces:

- **`Loginfy`** — The central orchestrator. Holds registered strategies, storage, session manager, and hooks.
- **`User`** — Represents an authenticated user with ID, email, password hash, roles, and metadata.
- **`Context`** — A request-scoped context carrying the HTTP request/response, Loginfy instance, and arbitrary key-value data.
- **`Strategy`** (interface) — Authentication strategy with `Name()` and `Authenticate()` methods.
- **`Storage`** (interface) — User persistence with CRUD operations.
- **`SessionManager`** (interface) — Session lifecycle: create, validate, destroy.
- **`Hooks`** — Callbacks for `OnLogin` and `OnRegister` events.
- **Errors** — Sentinel errors for common failure modes.

### 6.2 Strategies (`strategies/`)

Authentication strategy implementations:

- **`emailPassword`** — Validates email/password credentials against storage using bcrypt. Also provides `Register()` for user creation.

### 6.3 Sessions (`sessions/`)

Session manager implementations:

- **`jwt`** — Stateless JWT tokens signed with HMAC-SHA256. Supports `CreateSession`, `CreateSessionWithUser`, `ValidateSession`, `ValidateSessionWithClaims`, and `DestroySession`.

### 6.4 Storage (`storage/`)

Storage adapter implementations:

- **`memory`** — Thread-safe in-memory storage using `sync.RWMutex`. Suitable for development and testing.
- **`mongodb`** — Placeholder for MongoDB adapter (methods return `ErrNotImplemented`).

### 6.5 Authorization (`authorization/`)

- **`Authorizer`** — Manages role-permission mappings and policy functions.
  - `DefineRole(role, permissions...)` — Create roles with permissions
  - `HasPermission(user, permission)` — Check if any user role grants a permission
  - `AllowPolicy(action, fn)` — Register policy functions
  - `Can(user, action, resource)` — Evaluate a policy

### 6.6 Middleware (`middleware/`)

HTTP middleware for `net/http`:

- **`RequireAuth`** — Checks for `Authorization` header presence
- **`RequireAuthWithLoginfy`** — Validates JWT token, loads user into context
- **`RequireRole`** — Ensures user has at least one required role
- **`RequirePermission`** — Checks user metadata for specific permission

### 6.7 Utilities (`pkg/`)

- **`crypto`** — bcrypt hashing, password verification, secure token generation, constant-time comparison
- **`logger`** — Structured logging with levels, service loggers, sessions, colored/JSON output
- **`constants`** — Shared string constants
- **`status`** — HTTP status code helpers

## 7. User Flows

### 7.1 Registration Flow

1. Application creates a `Context` with email and password
2. `EmailPasswordStrategy.Register()` is called
3. Password is hashed with bcrypt
4. A unique user ID is generated
5. User is persisted via `Storage.CreateUser()`
6. `OnRegister` hook fires (if configured)
7. User object is returned

### 7.2 Authentication Flow

1. Application creates a `Context` with credentials
2. `Loginfy.Authenticate("email_password", ctx)` is called
3. Strategy fetches user from storage by email
4. Password is verified with bcrypt
5. User is stored in context
6. `OnLogin` hook fires (if configured)
7. `Loginfy.Login(user)` creates a JWT session token
8. Token is returned to the caller

### 7.3 Request Authorization Flow

1. HTTP request arrives with `Authorization: Bearer <token>` header
2. `RequireAuthWithLoginfy` middleware validates the JWT
3. User is loaded from storage and set in context
4. `RequireRole` or `RequirePermission` middleware checks access
5. Request proceeds to handler or is rejected with 403

## 8. API Surface

### Core

| Function/Method                           | Description                              |
|------------------------------------------|------------------------------------------|
| `core.New() *Loginfy`                    | Create new instance                      |
| `Loginfy.Use(Strategy)`                  | Register strategy                        |
| `Loginfy.SetStorage(Storage)`            | Set storage adapter                      |
| `Loginfy.SetSessionManager(SessionManager)` | Set session manager                   |
| `Loginfy.SetHooks(Hooks)`               | Set lifecycle hooks                      |
| `Loginfy.Authenticate(name, ctx)`        | Authenticate via named strategy          |
| `Loginfy.Login(user)`                    | Create session, return token             |
| `Loginfy.Logout(ctx, token)`            | Destroy session                          |
| `Loginfy.Mount()`                        | HTTP middleware for context injection     |

### Middleware

| Function                                  | Description                              |
|------------------------------------------|------------------------------------------|
| `RequireAuth(handler)`                   | Check Authorization header exists        |
| `RequireAuthWithLoginfy(loginfy)(handler)` | Validate JWT, load user               |
| `RequireRole(loginfy, roles...)(handler)` | Check user roles                        |
| `RequirePermission(loginfy, perm)(handler)` | Check user permission                 |

### Authorization

| Function/Method                           | Description                              |
|------------------------------------------|------------------------------------------|
| `authorization.New()`                    | Create authorizer                        |
| `Authorizer.DefineRole(role, perms...)`  | Define role with permissions             |
| `Authorizer.HasPermission(user, perm)`   | Check if user has permission via roles   |
| `Authorizer.AllowPolicy(action, fn)`     | Register policy function                 |
| `Authorizer.Can(user, action, resource)` | Evaluate policy                          |

## 9. Data Models

### User

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

### JWT Claims

```go
type Claims struct {
    UserID    string    `json:"user_id"`
    Email     string    `json:"email"`
    Roles     []string  `json:"roles,omitempty"`
    IssuedAt  time.Time `json:"iat"`
    ExpiresAt time.Time `json:"exp"`
}
```

## 10. Security Requirements

1. Passwords MUST be hashed with bcrypt before storage
2. JWT tokens MUST be signed with HMAC-SHA256
3. Token secrets MUST be at least 32 characters in production
4. Password fields MUST never be serialized in JSON responses
5. Token comparison MUST use constant-time algorithms
6. All cryptographic randomness MUST use `crypto/rand`
7. Error messages MUST NOT leak implementation details

## 11. Storage Interface Contract

```go
type Storage interface {
    CreateUser(user *User) error
    GetUserByEmail(email string) (*User, error)
    GetUserById(id string) (*User, error)
    UpdateUser(user *User) error
    DeleteUser(id string) error
}
```

All implementations must be safe for concurrent use.

## 12. Strategy Interface Contract

```go
type Strategy interface {
    Name() string
    Authenticate(ctx *Context) (*User, error)
}
```

Strategies access storage and other services through the `Context.Loginfy` reference.

## 13. Session Manager Interface Contract

```go
type SessionManager interface {
    CreateSession(userId string) (string, error)
    ValidateSession(ctx *Context, token string) (string, error)
    DestroySession(ctx *Context, token string) error
}
```

## 14. Logging Requirements

- All framework components MUST use the structured logger
- Log entries MUST include contextual fields (request ID, user ID, etc.)
- Authentication failures MUST be logged at WARN level
- Configuration errors MUST be logged at ERROR level
- Routine operations SHOULD be logged at DEBUG level
- Session logging MUST support file-based output for audit trails

## 15. Testing Requirements

- All exported functions MUST have unit tests
- Storage adapters MUST be tested for concurrent access safety
- JWT token generation and validation MUST be tested end-to-end
- Middleware MUST be tested with `httptest`
- Password hashing MUST be tested for correctness
- Target: >80% code coverage

## 16. Dependencies

| Dependency           | Purpose                 | Version   |
|---------------------|-------------------------|-----------|
| `golang.org/x/crypto` | bcrypt password hashing | v0.48.0+  |

The framework intentionally minimizes external dependencies.

## 17. Performance Requirements

- In-memory storage operations: <1ms per operation
- JWT token generation: <5ms
- JWT token validation: <2ms
- Middleware overhead: <1ms per request
- The framework should not be the bottleneck in any application

## 18. Compatibility

- Go 1.21+ required
- Compatible with `net/http` and any router that implements `http.Handler`
- No CGO dependencies
- Cross-platform (Linux, macOS, Windows)

## 19. Release Plan

| Version | Scope                                                        | Status    |
|---------|--------------------------------------------------------------|-----------|
| v0.1.0  | Core framework, email/password, JWT, memory storage, middleware | Current |
| v0.2.0  | OAuth strategies (Google, GitHub, Discord)                   | Planned   |
| v0.3.0  | Enhanced RBAC, permission inheritance, audit logging          | Planned   |
| v0.4.0  | MongoDB/PostgreSQL storage, rate limiting, 2FA, refresh tokens| Planned   |

## 20. Success Metrics

1. **Adoption** — GitHub stars, forks, and go module downloads
2. **Developer Experience** — Time from `go get` to working auth < 5 minutes
3. **Reliability** — Zero security vulnerabilities in released versions
4. **Performance** — All benchmarks meet or exceed requirements
5. **Community** — Active contributions and issue engagement
