package emailPassword

import (
	"errors"
	"fmt"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/pkg/constants"
	"github.com/AryanAg08/loginfy-go/pkg/crypto"
	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

var (
	ErrMissingCredentials = errors.New("email and password are required")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// EmailPasswordStrategy authenticates users via email and password
type EmailPasswordStrategy struct {
	log *logger.ServiceLogger
}

// New creates a new email/password strategy
func New() *EmailPasswordStrategy {
	return &EmailPasswordStrategy{
		log: logger.NewServiceLogger("email-password-strategy"),
	}
}

// Name returns the strategy name
func (s *EmailPasswordStrategy) Name() string {
	return constants.StrategyEmailPassword
}

// Authenticate validates email/password against the storage backend
func (s *EmailPasswordStrategy) Authenticate(ctx *core.Context) (*core.User, error) {
	sessionID := fmt.Sprintf("auth-%s", ctx.RequestID)
	sess, err := s.log.Logger().StartSession(sessionID)
	if err == nil {
		defer sess.End()
		sess.Info("authentication attempt started")
	}

	email := ctx.GetString("email")
	password := ctx.GetString("password")

	if email == "" || password == "" {
		s.log.Warn("missing credentials", map[string]interface{}{
			"request_id": ctx.RequestID,
		})
		return nil, ErrMissingCredentials
	}

	// Fetch user from storage
	storage := ctx.Loginfy.GetStorage()
	if storage == nil {
		s.log.Error("storage not configured", nil)
		return nil, core.ErrStorageNotSet
	}

	user, err := storage.GetUserByEmail(email)
	if err != nil {
		s.log.Warn("user not found", map[string]interface{}{
			"email":      email,
			"request_id": ctx.RequestID,
		})
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := crypto.VerifyPassword(password, user.Password); err != nil {
		s.log.Warn("password mismatch", map[string]interface{}{
			"email":      email,
			"request_id": ctx.RequestID,
		})
		return nil, ErrInvalidCredentials
	}

	if sess != nil {
		sess.Info("authentication successful", map[string]interface{}{
			"user_id": user.ID,
		})
	}
	s.log.Info("user authenticated", map[string]interface{}{
		"user_id":    user.ID,
		"email":      email,
		"request_id": ctx.RequestID,
	})

	return user, nil
}

// Register creates a new user with hashed password
func (s *EmailPasswordStrategy) Register(ctx *core.Context) (*core.User, error) {
	email := ctx.GetString("email")
	password := ctx.GetString("password")

	if email == "" || password == "" {
		return nil, ErrMissingCredentials
	}

	storage := ctx.Loginfy.GetStorage()
	if storage == nil {
		return nil, core.ErrStorageNotSet
	}

	hashedPassword, err := crypto.HashPassword(password)
	if err != nil {
		s.log.Error("failed to hash password", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	userID, _ := crypto.GenerateToken(16)
	user := &core.User{
		ID:       userID,
		Email:    email,
		Password: hashedPassword,
		Roles:    []string{"user"},
	}

	if err := storage.CreateUser(user); err != nil {
		s.log.Warn("user creation failed", map[string]interface{}{
			"email": email,
			"error": err.Error(),
		})
		return nil, err
	}

	s.log.Info("user registered", map[string]interface{}{
		"user_id": user.ID,
		"email":   email,
	})

	return user, nil
}
