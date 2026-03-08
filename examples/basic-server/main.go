package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/AryanAg08/loginfy.go/core"
	"github.com/AryanAg08/loginfy.go/middleware"
	"github.com/AryanAg08/loginfy.go/pkg/logger"
	"github.com/AryanAg08/loginfy.go/strategies/emailPassword"
)

func main() {
	// Initialize service logger for the main application
	appLogger := logger.NewServiceLogger("loginfy-app", logger.Config{
		Level:      logger.DEBUG,
		TimeFormat: time.RFC3339,
		LogDir:     "/tmp/lognify/app-logs",
		UseColor:   true,
		JSONOutput: false,
	})

	appLogger.Info("application starting", map[string]interface{}{
		"port": 8080,
		"env":  "development",
	})

	// Create Loginfy instance
	app := core.New()

	// Register email/password strategy
	emailPasswordStrategy := emailPassword.New()
	app.Use(emailPasswordStrategy)

	appLogger.Info("strategy registered", map[string]interface{}{
		"strategy": emailPasswordStrategy.Name(),
	})

	// Setup HTTP routes
	mux := http.NewServeMux()

	// Public endpoint - no auth required
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		appLogger.Debug("health check", map[string]interface{}{
			"remote": r.RemoteAddr,
		})
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"healthy"}`)
	})

	// Auth endpoint - uses the email/password strategy
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Create a unique request ID
		requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())

		// Create context
		ctx := &core.Context{
			Request:   r,
			Response:  w,
			Loginfy:   app,
			RequestID: requestID,
		}

		// Extract credentials from request
		email := r.FormValue("email")
		password := r.FormValue("password")
		ctx.Set("email", email)
		ctx.Set("password", password)

		appLogger.Info("login attempt", map[string]interface{}{
			"request_id": requestID,
			"email":      email,
		})

		// Authenticate using the strategy
		user, err := emailPasswordStrategy.Authenticate(ctx)
		if err != nil {
			appLogger.Error("authentication failed", map[string]interface{}{
				"request_id": requestID,
				"error":      err.Error(),
			})
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, `{"error":"%s"}`, err.Error())
			return
		}

		appLogger.Info("authentication successful", map[string]interface{}{
			"request_id": requestID,
			"user_id":    user.ID,
		})

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"user_id":"%s","email":"%s"}`, user.ID, user.Email)
	})

	// Protected endpoint - requires auth
	protectedHandler := middleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		appLogger.Info("protected resource accessed", map[string]interface{}{
			"path": r.URL.Path,
		})
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"message":"welcome to protected resource"}`)
	}))
	mux.Handle("/protected", protectedHandler)

	// Wrap the entire mux with HTTP logging middleware
	handler := appLogger.HTTPMiddleware(mux)

	// Start server
	addr := ":8080"
	appLogger.Info("server starting", map[string]interface{}{
		"address": addr,
	})

	if err := http.ListenAndServe(addr, handler); err != nil {
		appLogger.Fatal("server failed", map[string]interface{}{
			"error": err.Error(),
		})
	}
}
