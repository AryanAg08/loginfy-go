package twitter

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
	authURL     = "https://twitter.com/i/oauth2/authorize"
	tokenURL    = "https://api.twitter.com/2/oauth2/token"
	userInfoURL = "https://api.twitter.com/2/users/me?user.fields=id,name,username,profile_image_url"
)

type userInfoResponse struct {
	Data struct {
		ID              string `json:"id"`
		Name            string `json:"name"`
		Username        string `json:"username"`
		ProfileImageURL string `json:"profile_image_url"`
	} `json:"data"`
}

// TwitterProvider implements the OAuthProvider interface for Twitter/X OAuth 2.0 with PKCE.
type TwitterProvider struct {
	config *oauth2.Config
	log    *logger.ServiceLogger
}

// New creates a new TwitterProvider with the given OAuth configuration.
func New(config core.OAuthConfig) *TwitterProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"tweet.read", "users.read"}
	}

	return &TwitterProvider{
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
		log: logger.NewServiceLogger("twitter-oauth"),
	}
}

func (p *TwitterProvider) Name() string         { return constants.StrategyTwitter }
func (p *TwitterProvider) ProviderName() string { return "twitter" }

func (p *TwitterProvider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
		oauth2.S256ChallengeOption(state),
	)
}

func (p *TwitterProvider) Authenticate(ctx *core.Context) (*core.User, error) {
	return p.HandleCallback(ctx)
}

func (p *TwitterProvider) HandleCallback(ctx *core.Context) (*core.User, error) {
	code := ctx.Request.URL.Query().Get("code")
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}

	// Exchange with PKCE verifier
	codeVerifier := ctx.Request.URL.Query().Get("state")
	token, err := p.config.Exchange(ctx.Request.Context(), code,
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	)
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
		Roles: []string{"user"},
		Metadata: map[string]interface{}{
			"provider":    "twitter",
			"provider_id": info.Data.ID,
			"username":    info.Data.Username,
			"name":        info.Data.Name,
			"avatar_url":  info.Data.ProfileImageURL,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	storage := ctx.Loginfy.GetStorage()
	if storage != nil {
		userID, _ := crypto.GenerateToken(16)
		user.ID = userID
		if err := storage.CreateUser(user); err != nil {
			p.log.Error("failed to create user", map[string]interface{}{"error": err.Error()})
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	p.log.Info("user authenticated via twitter", map[string]interface{}{"username": info.Data.Username})
	return user, nil
}

func (p *TwitterProvider) fetchUserInfo(token *oauth2.Token) (*userInfoResponse, error) {
	client := p.config.Client(nil, token)
	resp, err := client.Get(userInfoURL)
	if err != nil {
		p.log.Error("failed to fetch user info", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("twitter user API returned %d: %s", resp.StatusCode, string(body))
	}

	var info userInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}
	return &info, nil
}
