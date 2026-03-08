package core

// OAuthProvider extends Strategy with OAuth-specific methods
type OAuthProvider interface {
	Strategy

	// AuthURL returns the URL to redirect the user to for authentication
	AuthURL(state string) string

	// HandleCallback processes the OAuth callback and returns the authenticated user
	HandleCallback(ctx *Context) (*User, error)

	// ProviderName returns the OAuth provider name (e.g., "google", "github")
	ProviderName() string
}

// OAuthConfig holds common OAuth configuration
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}
