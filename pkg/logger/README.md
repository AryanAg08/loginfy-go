# Lognify.go Logger Package

Production-ready structured logging system for Go applications with service-based tagging, session management, and automatic file generation.

---

## 📦 Package Structure

```
pkg/logger/
├── logger.go       # Core logger with 5 log levels, timestamps, service tags
├── session.go      # Session management with temp file generation
├── service.go      # Service-specific logger wrapper
└── logger_test.go  # Comprehensive test suite (10 tests)
```

---

## ✨ Features

- ✅ **5 Log Levels**: DEBUG, INFO, WARN, ERROR, FATAL
- ✅ **Timestamps**: RFC3339 format on every log entry
- ✅ **Service Tags**: Auto-tagged with service name (`service=auth-service`)
- ✅ **Caller Location**: Automatic `file:line` capture
- ✅ **Structured Fields**: Key-value metadata in every log
- ✅ **Session Temp Files**: Auto-generated `.log` files per session
- ✅ **Color Output**: Terminal-friendly ANSI colors
- ✅ **JSON Mode**: Production-ready JSON output for log aggregation
- ✅ **HTTP Middleware**: Built-in request/response logging
- ✅ **Concurrency-Safe**: Mutex-protected, safe for goroutines

---

## 🚀 Quick Start

### 1. Basic Logging

```go
package main

import "github.com/AryanAg08/loginfy-go/pkg/logger"

func main() {
    logger.Infof("Server started on port %d", 8080)
    
    logger.InfoMsg("User logged in", map[string]interface{}{
        "user_id": "u123",
        "ip": "192.168.1.1",
    })
}
```

**Output:**
```
[INFO] 2026-03-08T18:40:50+05:30 | service=default | main.go:5 | Server started on port 8080
[INFO] 2026-03-08T18:40:50+05:30 | service=default | main.go:7 | User logged in | user_id=u123 ip=192.168.1.1
```

---

### 2. Service-Specific Logger

```go
package main

import "github.com/AryanAg08/loginfy-go/pkg/logger"

func main() {
    apiLogger := logger.NewServiceLogger("api-gateway", logger.Config{
        Level: logger.DEBUG,
        UseColor: true,
    })
    
    apiLogger.Info("Request routed", map[string]interface{}{
        "path": "/api/users",
        "method": "GET",
    })
    
    apiLogger.Warn("High latency detected", map[string]interface{}{
        "duration_ms": 1500,
        "threshold": 1000,
    })
}
```

**Output:**
```
[INFO] 2026-03-08T18:40:50+05:30 | service=api-gateway | main.go:11 | Request routed | path=/api/users method=GET
[WARN] 2026-03-08T18:40:50+05:30 | service=api-gateway | main.go:16 | High latency detected | duration_ms=1500 threshold=1000
```

---

### 3. Session Logging with Temp Files

Sessions automatically create temporary log files in `/tmp/lognify/` (configurable).

```go
package main

import (
    "github.com/AryanAg08/loginfy-go/pkg/logger"
)

func main() {
    orderLogger := logger.NewServiceLogger("order-service")
    
    // Start a session - creates: order-service_order-123_20260308_184050.log
    sess, err := orderLogger.Logger().StartSession("order-123")
    if err != nil {
        panic(err)
    }
    defer sess.End() // Writes footer and closes file
    
    sess.Info("Order processing started", map[string]interface{}{
        "order_id": "order-123",
        "user_id": "u456",
    })
    
    // Process order...
    
    sess.Info("Order completed", map[string]interface{}{
        "total_usd": 99.99,
    })
}
```

**Temp File Created:** `/tmp/lognify/order-service_order-123_20260308_184050.log`

**File Contents:**
```
=== SESSION STARTED ===
Session ID : order-123
Service    : order-service
Started At : 2026-03-08T18:40:50+05:30
Log File   : /tmp/lognify/order-service_order-123_20260308_184050.log
========================

[INFO] 2026-03-08T18:40:50+05:30 | service=order-service | session=order-123 | main.go:15 | Order processing started | order_id=order-123 user_id=u456
[INFO] 2026-03-08T18:40:52+05:30 | service=order-service | session=order-123 | main.go:20 | Order completed | total_usd=99.99

========================
Session ID : order-123
Duration   : 2.145s
Ended At   : 2026-03-08T18:40:52+05:30
=== SESSION ENDED ===
```

---

### 4. HTTP Middleware

Automatically logs all incoming requests and responses.

```go
package main

import (
    "net/http"
    "github.com/AryanAg08/loginfy-go/pkg/logger"
)

func main() {
    apiLogger := logger.NewServiceLogger("api-server")
    
    mux := http.NewServeMux()
    mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"users":[]}`))
    })
    
    // Wrap with logging middleware
    handler := apiLogger.HTTPMiddleware(mux)
    
    http.ListenAndServe(":8080", handler)
}
```

**Output on each request:**
```
[INFO] 2026-03-08T18:40:50+05:30 | service=api-server | service.go:42 | request started | method=GET path=/users remote=127.0.0.1:52834
[INFO] 2026-03-08T18:40:50+05:30 | service=api-server | service.go:50 | request completed | method=GET path=/users status=200 duration=5ms
```

---

### 5. Using in Strategies/Packages

**File:** `strategies/emailPassword/emailPassword.go`

```go
package emailPassword

import (
    "github.com/AryanAg08/loginfy-go/core"
    "github.com/AryanAg08/loginfy-go/pkg/logger"
)

type EmailPasswordStrategy struct {
    log *logger.ServiceLogger
}

func New() *EmailPasswordStrategy {
    return &EmailPasswordStrategy{
        log: logger.NewServiceLogger("email-password-strategy"),
    }
}

func (s *EmailPasswordStrategy) Authenticate(ctx *core.Context) (*core.User, error) {
    // Start a session for this auth attempt
    sessionID := fmt.Sprintf("auth-%s", ctx.RequestID)
    sess, err := s.log.Logger().StartSession(sessionID)
    if err == nil {
        defer sess.End()
        sess.Info("authentication attempt started")
    }
    
    email := ctx.GetString("email")
    s.log.Info("authenticating user", map[string]interface{}{
        "email": email,
        "request_id": ctx.RequestID,
    })
    
    // ... authentication logic ...
    
    s.log.Info("authentication successful", map[string]interface{}{
        "user_id": user.ID,
    })
    
    return user, nil
}
```

---

## 🎨 Configuration Options

```go
logger.Config{
    Service:    "my-service",      // Service name (appears in every log)
    Level:      logger.DEBUG,      // Minimum log level (DEBUG/INFO/WARN/ERROR/FATAL)
    TimeFormat: time.RFC3339,      // Timestamp format
    LogDir:     "/tmp/lognify",    // Directory for session temp files
    UseColor:   true,              // ANSI color output for terminals
    JSONOutput: false,             // JSON format (for production log aggregation)
}
```

---

## 📊 JSON Output Mode

For production environments with log aggregation (ELK, Splunk, etc.):

```go
prodLogger := logger.NewServiceLogger("payment-service", logger.Config{
    Level:      logger.INFO,
    JSONOutput: true,
    UseColor:   false,
})

prodLogger.Info("Payment processed", map[string]interface{}{
    "transaction_id": "txn-999",
    "amount": 250.00,
    "currency": "USD",
})
```

**Output:**
```json
{"timestamp":"2026-03-08T18:40:50+05:30","level":"INFO","service":"payment-service","caller":"main.go:12","message":"Payment processed","transaction_id":"txn-999","amount":"250","currency":"USD"}
```

---

## 🧪 Testing

Run the comprehensive test suite:

```bash
cd /Users/aryanag/Documents/GitHub/Lognify.go
go test ./pkg/logger/ -v
```

**10 tests covering:**
- Basic logging output
- Log level filtering
- Structured fields
- JSON output
- Child loggers (ForService)
- Session creation
- Session file naming
- Duplicate session prevention
- Multiple active sessions
- Service logger wrapper

---

## 📁 Session Temp Files

### Naming Convention
```
<service>_<sessionID>_<timestamp>.log
```

**Examples:**
- `order-service_order-123_20260308_184050.log`
- `auth-service_auth-req-456_20260308_184051.log`
- `batch-processor_batch-item-1_20260308_184052.log`

### Default Location
```
/tmp/lognify/
```

Override with:
```go
logger.Config{
    LogDir: "/var/log/myapp",
}
```

---

## 🔥 Advanced Patterns

### Pattern 1: Multi-Service Application

```go
// Each service gets its own logger
authLogger := logger.NewServiceLogger("auth-service")
orderLogger := logger.NewServiceLogger("order-service")
paymentLogger := logger.NewServiceLogger("payment-service")

// Logs are automatically tagged
authLogger.Info("User authenticated") 
// → service=auth-service

orderLogger.Info("Order created")     
// → service=order-service
```

### Pattern 2: Request Tracing

```go
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    requestID := uuid.New().String()
    sess, _ := appLogger.Logger().StartSession(requestID)
    defer sess.End()
    
    sess.Info("Request started", map[string]interface{}{
        "method": r.Method,
        "path": r.URL.Path,
    })
    
    // Process request...
    
    sess.Info("Request completed")
}
```

### Pattern 3: Graceful Shutdown

```go
func main() {
    appLogger := logger.NewServiceLogger("app")
    defer appLogger.Logger().CloseAllSessions() // Close all open sessions
    
    // ... application logic ...
}
```

---

## 🆚 Comparison with Standard Library

| Feature | `log` (stdlib) | `pkg/logger` |
|---------|---------------|-------------|
| Log Levels | ❌ | ✅ 5 levels |
| Timestamps | ⚠️ Manual | ✅ Automatic |
| Service Tags | ❌ | ✅ Built-in |
| Structured Fields | ❌ | ✅ Key-value pairs |
| Session Files | ❌ | ✅ Automatic |
| Caller Location | ❌ | ✅ Auto-captured |
| JSON Output | ❌ | ✅ Configurable |
| HTTP Middleware | ❌ | ✅ Built-in |
| Concurrency-Safe | ⚠️ Partial | ✅ Full |

---

## 📚 API Reference

### Core Functions

```go
// Package-level (uses default logger)
logger.Infof(format string, args ...interface{})
logger.InfoMsg(msg string, fields ...map[string]interface{})
logger.Debugf / Debug / Warnf / Warn / Errorf / Error / Fatalf / FatalMsg

// Create service logger
logger.NewServiceLogger(service string, cfg ...logger.Config) *ServiceLogger

// Session management
logger.StartSession(sessionID string) (*Session, error)
logger.GetSession(sessionID string) (*Session, bool)
logger.ActiveSessions() int
logger.CloseAllSessions()
```

### ServiceLogger Methods

```go
sl := logger.NewServiceLogger("my-service")

sl.Debug(msg string, fields ...map[string]interface{})
sl.Info(msg string, fields ...map[string]interface{})
sl.Warn(msg string, fields ...map[string]interface{})
sl.Error(msg string, fields ...map[string]interface{})
sl.Fatal(msg string, fields ...map[string]interface{})

sl.Debugf(format string, args ...interface{})
sl.Infof(format string, args ...interface{})
// ... etc

sl.StartSession(sessionID string) (*Session, error)
sl.HTTPMiddleware(next http.Handler) http.Handler
sl.Logger() *logger.Logger
```

### Session Methods

```go
sess, _ := logger.StartSession("sess-123")

sess.Info(msg string, fields ...map[string]interface{})
sess.Debug / Warn / Error

sess.End() error // Closes file, logs duration

// Properties
sess.ID        string
sess.Service   string
sess.StartedAt time.Time
sess.FilePath  string
```

---

## 🛠️ Integration Examples

### With Gin Framework

```go
import (
    "github.com/gin-gonic/gin"
    "github.com/AryanAg08/loginfy-go/pkg/logger"
)

func main() {
    r := gin.New()
    appLogger := logger.NewServiceLogger("api")
    
    // Logging middleware
    r.Use(func(c *gin.Context) {
        appLogger.Info("Request", map[string]interface{}{
            "method": c.Request.Method,
            "path": c.Request.URL.Path,
        })
        c.Next()
    })
    
    r.Run(":8080")
}
```

### With gRPC

```go
import (
    "google.golang.org/grpc"
    "github.com/AryanAg08/loginfy-go/pkg/logger"
)

type Server struct {
    log *logger.ServiceLogger
}

func (s *Server) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
    sess, _ := s.log.Logger().StartSession(req.RequestId)
    defer sess.End()
    
    sess.Info("Creating user", map[string]interface{}{
        "email": req.Email,
    })
    
    // ... implementation ...
    
    return user, nil
}
```

---

## 🎯 Best Practices

1. **One logger per service/package**: Create a `ServiceLogger` in each package's init or constructor
2. **Use sessions for workflows**: Start a session for multi-step operations
3. **Always defer sess.End()**: Ensures proper cleanup
4. **Use structured fields**: Prefer `map[string]interface{}` over string concatenation
5. **Set appropriate log levels**: DEBUG for dev, INFO for production
6. **JSON in production**: Enable `JSONOutput: true` for log aggregation systems
7. **Don't log sensitive data**: Avoid passwords, tokens in logs

---

## 📄 License

Part of the Lognify.go authentication framework.
