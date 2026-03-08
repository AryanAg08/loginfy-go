package main

import (
	"fmt"
	"time"

	"github.com/AryanAg08/loginfy.go/core"
	"github.com/AryanAg08/loginfy.go/sessions/jwt"
	"github.com/AryanAg08/loginfy.go/storage/memory"
	"github.com/AryanAg08/loginfy.go/strategies/emailPassword"
)

func main() {
	// 1. Create the Loginfy instance
	app := core.New()

	// 2. Register the email/password strategy
	ep := emailPassword.New()
	app.Use(ep)

	// 3. Set up in-memory storage
	app.SetStorage(memory.New())

	// 4. Set up JWT session manager
	app.SetSessionManager(jwt.New(jwt.Config{
		Secret:     "example-secret-key-change-in-prod!",
		Expiration: 1 * time.Hour,
	}))

	// 5. Register a user
	regCtx := &core.Context{Loginfy: app, RequestID: "register"}
	regCtx.Set("email", "alice@example.com")
	regCtx.Set("password", "securePassword123")

	user, err := ep.Register(regCtx)
	if err != nil {
		fmt.Printf("Registration failed: %v\n", err)
		return
	}
	fmt.Printf("✅ User registered: %s (ID: %s)\n", user.Email, user.ID)

	// 6. Authenticate the user
	authCtx := &core.Context{Loginfy: app, RequestID: "login"}
	authCtx.Set("email", "alice@example.com")
	authCtx.Set("password", "securePassword123")

	user, err = app.Authenticate("email_password", authCtx)
	if err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
		return
	}
	fmt.Printf("✅ User authenticated: %s\n", user.Email)

	// 7. Create a JWT token
	token, err := app.Login(user)
	if err != nil {
		fmt.Printf("Session creation failed: %v\n", err)
		return
	}
	fmt.Printf("✅ JWT token created: %s...\n", token[:50])

	// 8. Validate the token
	validateCtx := &core.Context{Loginfy: app, RequestID: "validate"}
	sm := app.GetSessionManager()
	userID, err := sm.ValidateSession(validateCtx, token)
	if err != nil {
		fmt.Printf("Token validation failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Token validated — User ID: %s\n", userID)

	// 9. Test wrong password
	badCtx := &core.Context{Loginfy: app, RequestID: "bad-login"}
	badCtx.Set("email", "alice@example.com")
	badCtx.Set("password", "wrongpassword")

	_, err = app.Authenticate("email_password", badCtx)
	if err != nil {
		fmt.Printf("✅ Bad password correctly rejected: %v\n", err)
	}

	fmt.Println("\n🎉 Basic auth example complete!")
}
