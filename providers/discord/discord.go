package discord

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
	authURL     = "https://discord.com/api/oauth2/authorize"
	tokenURL    = "https://discord.com/api/oauth2/token"
	userInfoURL = "https://discord.com/api/users/@me"
)

type userInfo struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Email         string `json:"email"`
	Verified      bool   `json:"verified"`
	Avatar        string `json:"avatar"`
	GlobalName    string `json:"global_name"`
}

// DiscordProvider implements the OAuthProvider interface for Discord OAuth2.
type DiscordProvider struct {
	config *oauth2.Config
	log    *logger.ServiceLogger
}

// New creates a new DiscordProvider with the given OAuth configuration.
func New(config core.OAuthConfig) *DiscordProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"identify", "email"}
	}

	return &DiscordProvider{
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
		log: logger.NewServiceLogger("discord-oauth"),
	}
}

func (p *DiscordProvider) Name() string         { return constants.StrategyDiscord }
func (p *DiscordProvider) ProviderName() string  { return "discord" }

func (p *DiscordProvider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *DiscordProvider) Authenticate(ctx *core.Context) (*core.User, error) {
	return p.HandleCallback(ctx)
}

func (p *DiscordProvider) HandleCallback(ctx *core.Context) (*core.User, error) {
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

	avatarURL := ""
	if info.Avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", info.ID, info.Avatar)
	}

	now := time.Now()
	user := &core.User{
		Email: info.Email,
		Roles: []string{"user"},
		Metadata: map[string]interface{}{
			"provider":    "discord",
			"provider_id": info.ID,
			"username":    info.Username,
			"name":        info.GlobalName,
			"avatar_url":  avatarURL,
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
			p.log.Info("user updated from discord", map[string]interface{}{"user_id": existing.ID})
			return existing, nil
		}

		userID, _ := crypto.GenerateToken(16)
		user.ID = userID
		if err := storage.CreateUser(user); err != nil {
			p.log.Error("failed to create user", map[string]interface{}{"error": err.Error()})
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	p.log.Info("user authenticated via discord", map[string]interface{}{"email": info.Email})
	return user, nil
}

func (p *DiscordProvider) fetchUserInfo(token *oauth2.Token) (*userInfo, error) {
	client := p.config.Client(nil, token)
	resp, err := client.Get(userInfoURL)
	if err != nil {
		p.log.Error("failed to fetch user info", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("discord user API returned %d: %s", resp.StatusCode, string(body))
	}

	var info userInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}
	return &info, nil
}
