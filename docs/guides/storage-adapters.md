# Storage Adapters Guide

Storage adapters handle user persistence in Lognify.go. The framework defines a `Storage` interface, and you can use a built-in adapter or implement your own.

## Storage Interface

All adapters implement the `core.Storage` interface:

```go
type Storage interface {
    CreateUser(user *User) error
    GetUserByEmail(email string) (*User, error)
    GetUserById(id string) (*User, error)
    UpdateUser(user *User) error
    DeleteUser(id string) error
}
```

All implementations **must** be safe for concurrent use from multiple goroutines.

## Memory Adapter

The in-memory adapter stores users in a thread-safe map. It's ideal for development, testing, and prototyping.

### Usage

```go
import "github.com/AryanAg08/loginfy.go/storage/memory"

store := memory.New()
app.SetStorage(store)
```

### Features

- Thread-safe via `sync.RWMutex`
- O(1) lookup by ID, O(n) lookup by email
- `Count()` returns total number of stored users
- `Clear()` removes all users (useful in tests)

### Example

```go
store := memory.New()

// Create a user
user := &core.User{
    ID:       "user-1",
    Email:    "test@example.com",
    Password: "hashed-password",
    Roles:    []string{"user"},
}
err := store.CreateUser(user)

// Look up by email
found, err := store.GetUserByEmail("test@example.com")

// Look up by ID
found, err = store.GetUserById("user-1")

// Update
user.Roles = append(user.Roles, "admin")
err = store.UpdateUser(user)

// Delete
err = store.DeleteUser("user-1")

// Check count
fmt.Println(store.Count()) // 0
```

### Error Handling

```go
import "github.com/AryanAg08/loginfy.go/storage/memory"

switch err {
case memory.ErrUserAlreadyExists:
    // CreateUser called with duplicate email
case memory.ErrUserNotFound:
    // GetUserByEmail, GetUserById, UpdateUser, or DeleteUser
}
```

## MongoDB Adapter (Placeholder)

The MongoDB adapter is included as a placeholder for future implementation. All methods currently return `ErrNotImplemented`.

```go
import "github.com/AryanAg08/loginfy.go/storage/mongodb"

store := mongodb.New(mongodb.Config{
    ConnectionString: "mongodb://localhost:27017",
    Database:         "myapp",
})

// All methods return mongodb.ErrNotImplemented
err := store.CreateUser(user) // error: not implemented
```

This will be fully implemented in a future release.

## Creating a Custom Adapter

To use a different database, implement the `core.Storage` interface:

### Example: PostgreSQL Adapter

```go
package postgres

import (
    "database/sql"
    "github.com/AryanAg08/loginfy.go/core"
)

type PostgresStorage struct {
    db *sql.DB
}

func New(db *sql.DB) *PostgresStorage {
    return &PostgresStorage{db: db}
}

func (s *PostgresStorage) CreateUser(user *core.User) error {
    _, err := s.db.Exec(
        `INSERT INTO users (id, email, password, roles, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6)`,
        user.ID, user.Email, user.Password, user.Roles,
        user.CreatedAt, user.UpdatedAt,
    )
    return err
}

func (s *PostgresStorage) GetUserByEmail(email string) (*core.User, error) {
    user := &core.User{}
    err := s.db.QueryRow(
        `SELECT id, email, password, roles, created_at, updated_at
         FROM users WHERE email = $1`, email,
    ).Scan(&user.ID, &user.Email, &user.Password, &user.Roles,
        &user.CreatedAt, &user.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("user not found")
    }
    return user, err
}

func (s *PostgresStorage) GetUserById(id string) (*core.User, error) {
    user := &core.User{}
    err := s.db.QueryRow(
        `SELECT id, email, password, roles, created_at, updated_at
         FROM users WHERE id = $1`, id,
    ).Scan(&user.ID, &user.Email, &user.Password, &user.Roles,
        &user.CreatedAt, &user.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("user not found")
    }
    return user, err
}

func (s *PostgresStorage) UpdateUser(user *core.User) error {
    _, err := s.db.Exec(
        `UPDATE users SET email=$1, password=$2, roles=$3, updated_at=$4
         WHERE id = $5`,
        user.Email, user.Password, user.Roles, user.UpdatedAt, user.ID,
    )
    return err
}

func (s *PostgresStorage) DeleteUser(id string) error {
    _, err := s.db.Exec(`DELETE FROM users WHERE id = $1`, id)
    return err
}
```

### Using Your Custom Adapter

```go
db, _ := sql.Open("postgres", "postgres://localhost/myapp?sslmode=disable")
store := postgres.New(db)
app.SetStorage(store)
```

### Implementation Guidelines

1. **Thread Safety** — All methods must be safe for concurrent access
2. **Error Handling** — Return meaningful errors; use sentinel errors for common cases
3. **Email Uniqueness** — `CreateUser` should reject duplicate emails
4. **Not Found** — Return an error when a user doesn't exist
5. **Password Storage** — Store the hashed password as-is; never re-hash

## Next Steps

- [Getting Started](getting-started.md) — Quick start guide
- [Core API Reference](../api/core.md) — Storage interface details
- [Strategies Guide](strategies.md) — How strategies use storage
