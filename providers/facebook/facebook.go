package facebook

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/pkg/constants"
	"github.com/AryanAg08/loginfy-go/pkg/crypto"
	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

const (
	authURL     = "https://www.facebook.com/v18.0/dialog/oauth"
	tokenURL    = "https://graph.facebook.com/v18.0/oauth/access_token"
	userInfoURL = "https://graph.facebook.com/v18.0/me?fields=id,name,email,picture"
)

type userInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
}

// FacebookProvider implements the OAuthProvider interface for Facebook OAuth2.
type FacebookProvider struct {
	config *oauth2.Config
	log    *logger.ServiceLogger
}

// New creates a new FacebookProvider with the given OAuth configuration.
func New(config core.OAuthConfig) *FacebookProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"email", "public_profile"}
	}

	return &FacebookProvider{
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
		log: logger.NewServiceLogger("facebook-oauth"),
	}
}

func (p *FacebookProvider) Name() string         { return constants.StrategyFacebook }
func (p *FacebookProvider) ProviderName() string { return "facebook" }

func (p *FacebookProvider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *FacebookProvider) Authenticate(ctx *core.Context) (*core.User, error) {
	return p.HandleCallback(ctx)
}

func (p *FacebookProvider) HandleCallback(ctx *core.Context) (*core.User, error) {
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

	now := time.Now()
	user := &core.User{
		Email: info.Email,
		Roles: []string{"user"},
		Metadata: map[string]interface{}{
			"provider":    "facebook",
			"provider_id": info.ID,
			"name":        info.Name,
			"avatar_url":  info.Picture.Data.URL,
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
			p.log.Info("user updated from facebook", map[string]interface{}{"user_id": existing.ID})
			return existing, nil
		}

		userID, _ := crypto.GenerateToken(16)
		user.ID = userID
		if err := storage.CreateUser(user); err != nil {
			p.log.Error("failed to create user", map[string]interface{}{"error": err.Error()})
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	p.log.Info("user authenticated via facebook", map[string]interface{}{"email": info.Email})
	return user, nil
}

func (p *FacebookProvider) fetchUserInfo(token *oauth2.Token) (*userInfo, error) {
	client := p.config.Client(nil, token)
	resp, err := client.Get(userInfoURL)
	if err != nil {
		p.log.Error("failed to fetch user info", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("facebook graph API returned %d: %s", resp.StatusCode, string(body))
	}

	var info userInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}
	return &info, nil
}
