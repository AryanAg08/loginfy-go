package core

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
)

// Common errors
var (
	ErrStrategyNotFound       = errors.New("authentication strategy not found")
	ErrSessionManagerNotSet   = errors.New("session manager not configured")
	ErrStorageNotSet          = errors.New("storage not configured")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrInsufficientRole       = errors.New("insufficient role permissions")
	ErrInsufficientPermission = errors.New("insufficient permissions")
)

// Context keys for storing Loginfy context
type contextKey string

const loginfyContextKey contextKey = "loginfy_context"

// ContextWithLoginfy stores the Loginfy context in the request context
func ContextWithLoginfy(ctx context.Context, loginfyCtx *Context) context.Context {
	return context.WithValue(ctx, loginfyContextKey, loginfyCtx)
}

// LoginfyFromContext retrieves the Loginfy context from the request context
func LoginfyFromContext(ctx context.Context) (*Context, bool) {
	loginfyCtx, ok := ctx.Value(loginfyContextKey).(*Context)
	return loginfyCtx, ok
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
