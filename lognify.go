// Package lognify provides a plug-and-play authentication and authorization
// framework for Go applications. It supports multiple authentication strategies,
// OAuth providers, JWT and session management, RBAC and policy-based authorization,
// storage adapters, and middleware integration.
//
// Quick Start:
//
//	import lognify "github.com/AryanAg08/loginfy-go"
//
//	auth := lognify.New()
//	auth.Use(emailpassword.New())
//	auth.SetStorage(memory.New())
//	auth.SetSessionManager(jwt.New("your-secret", time.Hour))
//
// For more information, see https://github.com/AryanAg08/loginfy-go
package lognify

import (
	"github.com/AryanAg08/loginfy-go/authorization"
	"github.com/AryanAg08/loginfy-go/core"
)

// New creates a new Lognify authentication instance.
func New() *core.Loginfy {
	return core.New()
}

// NewAuthorization creates a new RBAC authorization manager.
func NewAuthorization() *authorization.Authorizer {
	return authorization.New()
}

// WithJWTSecret is a convenience option type for configuring JWT secret.
type Option func(*core.Loginfy)

// Configure applies options to a Loginfy instance.
func Configure(l *core.Loginfy, opts ...Option) {
	for _, opt := range opts {
		opt(l)
	}
}

// WithStorage returns an option that sets the storage adapter.
func WithStorage(s core.Storage) Option {
	return func(l *core.Loginfy) {
		l.SetStorage(s)
	}
}

// WithSessionManager returns an option that sets the session manager.
func WithSessionManager(sm core.SessionManager) Option {
	return func(l *core.Loginfy) {
		l.SetSessionManager(sm)
	}
}
