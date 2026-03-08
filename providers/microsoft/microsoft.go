package microsoft

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

const userInfoURL = "https://graph.microsoft.com/v1.0/me"

type userInfo struct {
	ID                string `json:"id"`
	DisplayName       string `json:"displayName"`
	Mail              string `json:"mail"`
	UserPrincipalName string `json:"userPrincipalName"`
}

// MicrosoftProvider implements the OAuthProvider interface for Microsoft Identity Platform.
type MicrosoftProvider struct {
	config *oauth2.Config
	log    *logger.ServiceLogger
}

// New creates a new MicrosoftProvider with the given OAuth configuration.
// An optional tenantID can be passed; defaults to "common" if empty.
func New(config core.OAuthConfig, tenantID ...string) *MicrosoftProvider {
	tenant := "common"
	if len(tenantID) > 0 && tenantID[0] != "" {
		tenant = tenantID[0]
	}

	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile", "User.Read"}
	}

	return &MicrosoftProvider{
		config: &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes:       scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenant),
				TokenURL: fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant),
			},
		},
		log: logger.NewServiceLogger("microsoft-oauth"),
	}
}

func (p *MicrosoftProvider) Name() string         { return constants.StrategyMicrosoft }
func (p *MicrosoftProvider) ProviderName() string  { return "microsoft" }

func (p *MicrosoftProvider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *MicrosoftProvider) Authenticate(ctx *core.Context) (*core.User, error) {
	return p.HandleCallback(ctx)
}

func (p *MicrosoftProvider) HandleCallback(ctx *core.Context) (*core.User, error) {
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

	email := info.Mail
	if email == "" {
		email = info.UserPrincipalName
	}

	now := time.Now()
	user := &core.User{
		Email: email,
		Roles: []string{"user"},
		Metadata: map[string]interface{}{
			"provider":    "microsoft",
			"provider_id": info.ID,
			"name":        info.DisplayName,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	storage := ctx.Loginfy.GetStorage()
	if storage != nil {
		existing, err := storage.GetUserByEmail(email)
		if err == nil && existing != nil {
			existing.Metadata = user.Metadata
			existing.UpdatedAt = now
			_ = storage.UpdateUser(existing)
			p.log.Info("user updated from microsoft", map[string]interface{}{"user_id": existing.ID})
			return existing, nil
		}

		userID, _ := crypto.GenerateToken(16)
		user.ID = userID
		if err := storage.CreateUser(user); err != nil {
			p.log.Error("failed to create user", map[string]interface{}{"error": err.Error()})
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	p.log.Info("user authenticated via microsoft", map[string]interface{}{"email": email})
	return user, nil
}

func (p *MicrosoftProvider) fetchUserInfo(token *oauth2.Token) (*userInfo, error) {
	client := p.config.Client(nil, token)
	resp, err := client.Get(userInfoURL)
	if err != nil {
		p.log.Error("failed to fetch user info", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("microsoft graph API returned %d: %s", resp.StatusCode, string(body))
	}

	var info userInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}
	return &info, nil
}
