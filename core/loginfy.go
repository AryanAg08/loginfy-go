package core

import (
	"net/http"

	"github.com/AryanAg08/loginfy.go/pkg/logger"
)

// Loginfy is the main authentication framework instance
type Loginfy struct {
	strategies map[string]Strategy
	storage    Storage
	session    SessionManager
	hooks      Hooks
	log        *logger.ServiceLogger
}

// New creates a new Loginfy instance
func New() *Loginfy {
	return &Loginfy{
		strategies: make(map[string]Strategy),
		log:        logger.NewServiceLogger("loginfy"),
	}
}

// Use registers an authentication strategy
func (l *Loginfy) Use(strategy Strategy) {
	l.strategies[strategy.Name()] = strategy
	l.log.Info("strategy registered", map[string]interface{}{
		"strategy": strategy.Name(),
	})
}

// SetStorage sets the storage adapter
func (l *Loginfy) SetStorage(storage Storage) {
	l.storage = storage
	l.log.Info("storage adapter set", nil)
}

// GetStorage returns the storage adapter
func (l *Loginfy) GetStorage() Storage {
	return l.storage
}

// SetSessionManager sets the session manager
func (l *Loginfy) SetSessionManager(session SessionManager) {
	l.session = session
	l.log.Info("session manager set", nil)
}

// GetSessionManager returns the session manager
func (l *Loginfy) GetSessionManager() SessionManager {
	return l.session
}

// SetHooks sets the authentication hooks
func (l *Loginfy) SetHooks(hooks Hooks) {
	l.hooks = hooks
	l.log.Info("hooks configured", nil)
}

// GetStrategy retrieves a registered strategy by name
func (l *Loginfy) GetStrategy(name string) (Strategy, bool) {
	strategy, ok := l.strategies[name]
	return strategy, ok
}

// Authenticate performs authentication using the specified strategy
func (l *Loginfy) Authenticate(strategyName string, ctx *Context) (*User, error) {
	strategy, ok := l.strategies[strategyName]
	if !ok {
		l.log.Error("strategy not found", map[string]interface{}{
			"strategy":   strategyName,
			"request_id": ctx.RequestID,
		})
		return nil, ErrStrategyNotFound
	}

	l.log.Info("authenticating with strategy", map[string]interface{}{
		"strategy":   strategyName,
		"request_id": ctx.RequestID,
	})

	user, err := strategy.Authenticate(ctx)
	if err != nil {
		l.log.Warn("authentication failed", map[string]interface{}{
			"strategy":   strategyName,
			"error":      err.Error(),
			"request_id": ctx.RequestID,
		})
		return nil, err
	}

	// Store user in context
	ctx.SetUser(user)

	// Call login hook if set
	if l.hooks.OnLogin != nil {
		l.hooks.OnLogin(user)
	}

	l.log.Info("authentication successful", map[string]interface{}{
		"strategy":   strategyName,
		"user_id":    user.ID,
		"request_id": ctx.RequestID,
	})

	return user, nil
}

// Mount returns an HTTP middleware that integrates Loginfy with your HTTP framework
// This middleware creates a Loginfy context for each request
func (l *Loginfy) Mount() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a Loginfy context
			ctx := &Context{
				Request:   r,
				Response:  w,
				Loginfy:   l,
				RequestID: generateRequestID(),
			}

			// Store context in request context for access in handlers
			r = r.WithContext(ContextWithLoginfy(r.Context(), ctx))

			l.log.Debug("request context created", map[string]interface{}{
				"request_id": ctx.RequestID,
				"path":       r.URL.Path,
				"method":     r.Method,
			})

			next.ServeHTTP(w, r)
		})
	}
}

// Login creates a session for a user after successful authentication
func (l *Loginfy) Login(user *User) (string, error) {
	if l.session == nil {
		l.log.Error("session manager not configured", map[string]interface{}{
			"user_id": user.ID,
		})
		return "", ErrSessionManagerNotSet
	}

	token, err := l.session.CreateSession(user.ID)
	if err != nil {
		l.log.Error("failed to create session", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
		})
		return "", err
	}

	l.log.Info("session created for user", map[string]interface{}{
		"user_id": user.ID,
	})

	return token, nil
}

// Logout destroys a user's session
func (l *Loginfy) Logout(ctx *Context, token string) error {
	if l.session == nil {
		l.log.Error("session manager not configured", map[string]interface{}{
			"request_id": ctx.RequestID,
		})
		return ErrSessionManagerNotSet
	}

	err := l.session.DestroySession(ctx, token)
	if err != nil {
		l.log.Error("failed to destroy session", map[string]interface{}{
			"error":      err.Error(),
			"request_id": ctx.RequestID,
		})
		return err
	}

	l.log.Info("session destroyed", map[string]interface{}{
		"request_id": ctx.RequestID,
	})

	return nil
}
