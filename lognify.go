// Package lognify provides a plug-and-play authentication and authorization
// framework for Go applications. It supports multiple authentication strategies,
// OAuth providers, JWT and session management, RBAC and policy-based authorization,
// storage adapters, and middleware integration.
//
// Quick Start:
//
//	auth := lognify.New()
//	auth.Use(emailpassword.New())
//	auth.SetStorage(memory.New())
//	auth.SetSessionManager(jwt.New("your-secret", time.Hour))
//
// For more information, see https://github.com/AryanAg08/loginfy-go
package lognify

import (
	"github.com/AryanAg08/loginfy-go/core"
)

// New creates a new Lognify authentication instance.
func New() *core.Loginfy {
	return core.New()
}
