package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AryanAg08/loginfy.go/core"
	"github.com/AryanAg08/loginfy.go/middleware"
	"github.com/AryanAg08/loginfy.go/sessions/jwt"
	"github.com/AryanAg08/loginfy.go/storage/memory"
	"github.com/AryanAg08/loginfy.go/strategies/emailPassword"
)

func main() {
	// Setup Loginfy
	app := core.New()
	ep := emailPassword.New()
	app.Use(ep)
	app.SetStorage(memory.New())
	app.SetSessionManager(jwt.New(jwt.Config{
		Secret:     "middleware-demo-secret-key-32chars!",
		Expiration: 1 * time.Hour,
	}))

	// Pre-register users
	registerUser(app, ep, "admin@example.com", "admin123", []string{"admin", "user"})
	registerUser(app, ep, "user@example.com", "user123", []string{"user"})

	// Routes
	mux := http.NewServeMux()

	// Public: health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"healthy"}`)
	})

	// Public: login endpoint — returns JWT token
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("email")
		password := r.URL.Query().Get("password")

		ctx := &core.Context{
			Request:   r,
			Response:  w,
			Loginfy:   app,
			RequestID: fmt.Sprintf("req-%d", time.Now().UnixNano()),
		}
		ctx.Set("email", email)
		ctx.Set("password", password)

		user, err := app.Authenticate("email_password", ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusUnauthorized)
			return
		}

		token, err := app.Login(user)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"token":"%s","email":"%s","roles":%q}`, token, user.Email, user.Roles)
	})

	// Protected: requires valid JWT
	mux.Handle("/api/profile", middleware.RequireAuthWithLoginfy(app)(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			loginfyCtx, _ := core.LoginfyFromContext(r.Context())
			user, _ := loginfyCtx.GetUser()

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"email":"%s","roles":%q}`, user.Email, user.Roles)
		}),
	))

	// Admin only: requires JWT + "admin" role
	mux.Handle("/api/admin", middleware.RequireAuthWithLoginfy(app)(
		middleware.RequireRole(app, "admin")(
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintln(w, `{"message":"welcome to the admin panel"}`)
			}),
		),
	))

	// Mount Loginfy context middleware
	handler := app.Mount()(mux)

	addr := ":8080"
	fmt.Println("🚀 Middleware demo server starting on", addr)
	fmt.Println()
	fmt.Println("Try these commands:")
	fmt.Println("  curl http://localhost:8080/health")
	fmt.Println("  curl 'http://localhost:8080/login?email=admin@example.com&password=admin123'")
	fmt.Println("  curl -H 'Authorization: Bearer <token>' http://localhost:8080/api/profile")
	fmt.Println("  curl -H 'Authorization: Bearer <token>' http://localhost:8080/api/admin")

	log.Fatal(http.ListenAndServe(addr, handler))
}

func registerUser(app *core.Loginfy, ep *emailPassword.EmailPasswordStrategy, email, password string, roles []string) {
	ctx := &core.Context{Loginfy: app, RequestID: "setup"}
	ctx.Set("email", email)
	ctx.Set("password", password)

	user, err := ep.Register(ctx)
	if err != nil {
		log.Fatalf("Failed to register %s: %v", email, err)
	}
	// Update roles
	user.Roles = roles
	if err := app.GetStorage().UpdateUser(user); err != nil {
		log.Fatalf("Failed to update roles for %s: %v", email, err)
	}
	fmt.Printf("Registered: %s (roles: %v)\n", email, roles)
}
