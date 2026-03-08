package middleware

import (
	"net/http"
	"strings"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/pkg/constants"
	"github.com/AryanAg08/loginfy-go/pkg/logger"
	"github.com/AryanAg08/loginfy-go/pkg/status"
)

var log = logger.NewServiceLogger("auth-middleware")

// RequireAuth middleware ensures that a request has a valid authentication token
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")

			if token == "" {
				log.Warn("authentication failed: missing token", map[string]interface{}{
					"path":   r.URL.Path,
					"method": r.Method,
					"remote": r.RemoteAddr,
				})
				http.Error(w, constants.AuthUnauthorized, status.StatusUnauthorized())
				return
			}

			log.Debug("authentication check passed", map[string]interface{}{
				"path": r.URL.Path,
			})

			next.ServeHTTP(w, r)
		})
}

// RequireAuthWithLoginfy middleware validates JWT token and loads user into context
// This requires Loginfy to be configured with a session manager and storage
func RequireAuthWithLoginfy(loginfy *core.Loginfy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get Loginfy context from request
			loginfyCtx, ok := core.LoginfyFromContext(r.Context())
			if !ok {
				log.Error("loginfy context not found - ensure Mount() middleware is used", map[string]interface{}{
					"path": r.URL.Path,
				})
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				log.Warn("authentication failed: missing token", map[string]interface{}{
					"path":       r.URL.Path,
					"request_id": loginfyCtx.RequestID,
				})
				http.Error(w, constants.AuthUnauthorized, status.StatusUnauthorized())
				return
			}

			// Remove "Bearer " prefix if present
			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate session
			sessionManager := loginfy.GetSessionManager()
			if sessionManager == nil {
				log.Error("session manager not configured", map[string]interface{}{
					"path":       r.URL.Path,
					"request_id": loginfyCtx.RequestID,
				})
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			userID, err := sessionManager.ValidateSession(loginfyCtx, token)
			if err != nil {
				log.Warn("session validation failed", map[string]interface{}{
					"error":      err.Error(),
					"path":       r.URL.Path,
					"request_id": loginfyCtx.RequestID,
				})
				http.Error(w, constants.AuthUnauthorized, status.StatusUnauthorized())
				return
			}

			// Load user from storage
			storage := loginfy.GetStorage()
			if storage == nil {
				log.Error("storage not configured", map[string]interface{}{
					"path":       r.URL.Path,
					"request_id": loginfyCtx.RequestID,
				})
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			user, err := storage.GetUserById(userID)
			if err != nil {
				log.Warn("failed to load user", map[string]interface{}{
					"error":      err.Error(),
					"user_id":    userID,
					"path":       r.URL.Path,
					"request_id": loginfyCtx.RequestID,
				})
				http.Error(w, constants.AuthUnauthorized, status.StatusUnauthorized())
				return
			}

			// Store user in context
			loginfyCtx.SetUser(user)

			log.Debug("user authenticated successfully", map[string]interface{}{
				"user_id":    user.ID,
				"email":      user.Email,
				"path":       r.URL.Path,
				"request_id": loginfyCtx.RequestID,
			})

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole middleware ensures the authenticated user has at least one of the specified roles
func RequireRole(loginfy *core.Loginfy, roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get Loginfy context from request
			loginfyCtx, ok := core.LoginfyFromContext(r.Context())
			if !ok {
				log.Error("loginfy context not found", map[string]interface{}{
					"path": r.URL.Path,
				})
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Get user from context
			user, ok := loginfyCtx.GetUser()
			if !ok {
				log.Warn("role check failed: no authenticated user", map[string]interface{}{
					"path":       r.URL.Path,
					"request_id": loginfyCtx.RequestID,
				})
				http.Error(w, "Forbidden: Authentication required", status.StatusForbidden())
				return
			}

			// Check if user has any of the required roles
			if !user.HasAnyRole(roles...) {
				log.Warn("role check failed: insufficient permissions", map[string]interface{}{
					"user_id":        user.ID,
					"user_roles":     user.Roles,
					"required_roles": roles,
					"path":           r.URL.Path,
					"request_id":     loginfyCtx.RequestID,
				})
				http.Error(w, "Forbidden: Insufficient permissions", status.StatusForbidden())
				return
			}

			log.Debug("role check passed", map[string]interface{}{
				"user_id":    user.ID,
				"user_roles": user.Roles,
				"path":       r.URL.Path,
				"request_id": loginfyCtx.RequestID,
			})

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission middleware checks if user has specific permission in metadata
// Permissions are stored in user.Metadata["permissions"] as []string
func RequirePermission(loginfy *core.Loginfy, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get Loginfy context from request
			loginfyCtx, ok := core.LoginfyFromContext(r.Context())
			if !ok {
				log.Error("loginfy context not found", map[string]interface{}{
					"path": r.URL.Path,
				})
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			// Get user from context
			user, ok := loginfyCtx.GetUser()
			if !ok {
				log.Warn("permission check failed: no authenticated user", map[string]interface{}{
					"path":       r.URL.Path,
					"request_id": loginfyCtx.RequestID,
				})
				http.Error(w, "Forbidden: Authentication required", status.StatusForbidden())
				return
			}

			// Check permissions in metadata
			if user.Metadata == nil {
				log.Warn("permission check failed: no metadata", map[string]interface{}{
					"user_id":             user.ID,
					"required_permission": permission,
					"path":                r.URL.Path,
					"request_id":          loginfyCtx.RequestID,
				})
				http.Error(w, "Forbidden: Insufficient permissions", status.StatusForbidden())
				return
			}

			// Get permissions from metadata
			permissionsRaw, ok := user.Metadata["permissions"]
			if !ok {
				log.Warn("permission check failed: no permissions in metadata", map[string]interface{}{
					"user_id":             user.ID,
					"required_permission": permission,
					"path":                r.URL.Path,
					"request_id":          loginfyCtx.RequestID,
				})
				http.Error(w, "Forbidden: Insufficient permissions", status.StatusForbidden())
				return
			}

			// Convert to string slice
			permissions, ok := permissionsRaw.([]string)
			if !ok {
				// Try to convert from interface slice
				if permSlice, ok := permissionsRaw.([]interface{}); ok {
					permissions = make([]string, 0, len(permSlice))
					for _, p := range permSlice {
						if pStr, ok := p.(string); ok {
							permissions = append(permissions, pStr)
						}
					}
				}
			}

			// Check if user has the required permission
			hasPermission := false
			for _, p := range permissions {
				if p == permission {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				log.Warn("permission check failed: permission not found", map[string]interface{}{
					"user_id":             user.ID,
					"user_permissions":    permissions,
					"required_permission": permission,
					"path":                r.URL.Path,
					"request_id":          loginfyCtx.RequestID,
				})
				http.Error(w, "Forbidden: Insufficient permissions", status.StatusForbidden())
				return
			}

			log.Debug("permission check passed", map[string]interface{}{
				"user_id":    user.ID,
				"permission": permission,
				"path":       r.URL.Path,
				"request_id": loginfyCtx.RequestID,
			})

			next.ServeHTTP(w, r)
		})
	}
}
