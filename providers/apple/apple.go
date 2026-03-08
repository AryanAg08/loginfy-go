package apple

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/pkg/constants"
	loginfyCrypto "github.com/AryanAg08/loginfy-go/pkg/crypto"
	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

const (
	authURL  = "https://appleid.apple.com/auth/authorize"
	tokenURL = "https://appleid.apple.com/auth/token"
)

// AppleConfig extends OAuthConfig with Apple-specific fields.
type AppleConfig struct {
	core.OAuthConfig
	TeamID     string
	KeyID      string
	PrivateKey string // PEM-encoded ECDSA private key
}

// idTokenClaims represents the claims in Apple's ID token.
type idTokenClaims struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
}

// AppleProvider implements the OAuthProvider interface for Apple Sign In.
type AppleProvider struct {
	config     *oauth2.Config
	teamID     string
	keyID      string
	privateKey *ecdsa.PrivateKey
	log        *logger.ServiceLogger
}

// New creates a new AppleProvider with the given Apple configuration.
func New(config AppleConfig) *AppleProvider {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{"name", "email"}
	}

	var privKey *ecdsa.PrivateKey
	block, _ := pem.Decode([]byte(config.PrivateKey))
	if block != nil {
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err == nil {
			if k, ok := key.(*ecdsa.PrivateKey); ok {
				privKey = k
			}
		}
	}

	return &AppleProvider{
		config: &oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: "", // generated dynamically
			RedirectURL:  config.RedirectURL,
			Scopes:       scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  authURL,
				TokenURL: tokenURL,
			},
		},
		teamID:     config.TeamID,
		keyID:      config.KeyID,
		privateKey: privKey,
		log:        logger.NewServiceLogger("apple-oauth"),
	}
}

func (p *AppleProvider) Name() string         { return constants.StrategyApple }
func (p *AppleProvider) ProviderName() string { return "apple" }

func (p *AppleProvider) AuthURL(state string) string {
	return p.config.AuthCodeURL(state, oauth2.SetAuthURLParam("response_mode", "form_post"))
}

func (p *AppleProvider) Authenticate(ctx *core.Context) (*core.User, error) {
	return p.HandleCallback(ctx)
}

func (p *AppleProvider) HandleCallback(ctx *core.Context) (*core.User, error) {
	code := ctx.Request.URL.Query().Get("code")
	if code == "" {
		// Apple uses form_post, so check form values too
		if err := ctx.Request.ParseForm(); err == nil {
			code = ctx.Request.FormValue("code")
		}
	}
	if code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}

	// Generate client secret JWT
	secret, err := p.generateClientSecret()
	if err != nil {
		p.log.Error("failed to generate client secret", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("failed to generate client secret: %w", err)
	}
	p.config.ClientSecret = secret

	token, err := p.config.Exchange(ctx.Request.Context(), code)
	if err != nil {
		p.log.Error("token exchange failed", map[string]interface{}{"error": err.Error()})
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}

	// Extract user info from ID token
	idToken, ok := token.Extra("id_token").(string)
	if !ok || idToken == "" {
		return nil, fmt.Errorf("missing id_token in response")
	}

	claims, err := p.parseIDToken(idToken)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &core.User{
		Email: claims.Email,
		Roles: []string{"user"},
		Metadata: map[string]interface{}{
			"provider":    "apple",
			"provider_id": claims.Sub,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	storage := ctx.Loginfy.GetStorage()
	if storage != nil {
		existing, err := storage.GetUserByEmail(claims.Email)
		if err == nil && existing != nil {
			existing.Metadata = user.Metadata
			existing.UpdatedAt = now
			_ = storage.UpdateUser(existing)
			p.log.Info("user updated from apple", map[string]interface{}{"user_id": existing.ID})
			return existing, nil
		}

		userID, _ := loginfyCrypto.GenerateToken(16)
		user.ID = userID
		if err := storage.CreateUser(user); err != nil {
			p.log.Error("failed to create user", map[string]interface{}{"error": err.Error()})
			return nil, fmt.Errorf("failed to create user: %w", err)
		}
	}

	p.log.Info("user authenticated via apple", map[string]interface{}{"email": claims.Email})
	return user, nil
}

// generateClientSecret creates a signed JWT for Apple's token endpoint.
func (p *AppleProvider) generateClientSecret() (string, error) {
	if p.privateKey == nil {
		return "", fmt.Errorf("private key not configured")
	}

	now := time.Now()
	exp := now.Add(5 * time.Minute)

	header := map[string]interface{}{
		"alg": "ES256",
		"kid": p.keyID,
	}
	payload := map[string]interface{}{
		"iss": p.teamID,
		"iat": now.Unix(),
		"exp": exp.Unix(),
		"aud": "https://appleid.apple.com",
		"sub": p.config.ClientID,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerB64 + "." + payloadB64

	h := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, p.privateKey, h[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	// Encode r and s as fixed-size 32-byte big-endian integers
	curveBits := p.privateKey.Curve.Params().BitSize
	keyBytes := curveBits / 8
	rBytes := r.Bytes()
	sBytes := s.Bytes()
	sig := make([]byte, 2*keyBytes)
	copy(sig[keyBytes-len(rBytes):keyBytes], rBytes)
	copy(sig[2*keyBytes-len(sBytes):], sBytes)

	sigB64 := base64.RawURLEncoding.EncodeToString(sig)
	return signingInput + "." + sigB64, nil
}

// parseIDToken extracts claims from an Apple ID token (JWT) without full verification.
func (p *AppleProvider) parseIDToken(idToken string) (*idTokenClaims, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid id_token format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode id_token payload: %w", err)
	}

	var claims idTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to parse id_token claims: %w", err)
	}
	return &claims, nil
}
