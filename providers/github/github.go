package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/AryanAg08/loginfy.go/core"
	"github.com/AryanAg08/loginfy.go/pkg/constants"
	"github.com/AryanAg08/loginfy.go/pkg/crypto"
	"github.com/AryanAg08/loginfy.go/pkg/logger"
)

const (
	authURL  = "https://github.com/login/oauth/authorize"
	tokenURL = "https://github.com/login/oauth/access_token"
	userURL  = "https://api.github.com/user"
	emailURL = "https://api.github.com/user/emails"
)

type userInfo struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type emailInfo struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

// GitHubProvider implements the OAuthProvider interface for GitHub OAuth2.
type GitHubProvider struct {
	config *oauth2.Config
	log    *logger.ServiceLogger
}

// New creates a new GitHubProvider with the given OAuth configuration.
func New(config core.OAuthConfig) *GitHubProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"user:email", "read:user"}
	}

	return &GitHubProvider{
		config: &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes:       scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
		},
		log: logger.NewServiceLogger("github-oauth"),
	}
}

func (p *GitHubProvider) Name() string         { return constants.StrategyGitHub }
func (p *GitHubProvider) ProviderName() string  { return "github" }

func (p *GitHubProvider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *GitHubProvider) Authenticate(ctx *core.Context) (*core.User, error) {
	return p.HandleCallback(ctx)
}

func (p *GitHubProvider) HandleCallback(ctx *core.Context) (*core.User, error) {
	code := ctx.Request.URL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}

	token, err := p.config.Exchange(ctx.Request.Context(), code)
	if err != nil {
		p.log.Error("token exchange failed", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	info, err := p.fetchUserInfo(token)
	if err != nil {
		return nil, err
	}

	// If email is empty, fetch from emails API
	if info.Email == "" {
		email, err := p.fetchPrimaryEmail(token)
		if err == nil {
			info.Email = email
		}
	}

	now := time.Now()
	user := &core.User{
		Email: info.Email,
		Roles: []string{"user"},
		Metadata: map[string]interface{}{
			"provider":    "github",
			"provider_id": fmt.Sprintf("%d", info.ID),
			"username":    info.Login,
			"name":        info.Name,
			"avatar_url":  info.AvatarURL,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	storage := ctx.Loginfy.GetStorage()
	if storage != nil {
		existing, err := storage.GetUserByEmail(info.Email)
		if err == nil && existing != nil {
			existing.Metadata = user.Metadata
			existing.UpdatedAt = now
			_ = storage.UpdateUser(existing)
			p.log.Info("user updated from github", map[string]interface{}{"user_id": existing.ID})
			return existing, nil
		}

		userID, _ := crypto.GenerateToken(16)
		user.ID = userID
		if err := storage.CreateUser(user); err != nil {
			p.log.Error("failed to create user", map[string]interface{}{"error": err.Error()})
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	p.log.Info("user authenticated via github", map[string]interface{}{"email": info.Email})
	return user, nil
}

func (p *GitHubProvider) fetchUserInfo(token *oauth2.Token) (*userInfo, error) {
	client := p.config.Client(nil, token)
	resp, err := client.Get(userURL)
	if err != nil {
		p.log.Error("failed to fetch user info", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github user API returned %d: %s", resp.StatusCode, string(body))
	}

	var info userInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}
	return &info, nil
}

func (p *GitHubProvider) fetchPrimaryEmail(token *oauth2.Token) (string, error) {
	client := p.config.Client(nil, token)
	resp, err := client.Get(emailURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github email API returned %d", resp.StatusCode)
	}

	var emails []emailInfo
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("failed to decode emails: %w", err)
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	if len(emails) > 0 {
		return emails[0].Email, nil
	}
	return "", fmt.Errorf("no email found")
}
