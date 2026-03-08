package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/AryanAg08/loginfy-go/core"
	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidSignature = errors.New("invalid token signature")
)

// JWTSessionManager manages JWT-based sessions
type JWTSessionManager struct {
	secret     []byte
	expiration time.Duration
	log        *logger.ServiceLogger
}

// Config holds JWT configuration options
type Config struct {
	Secret     string        // Secret key for signing tokens
	Expiration time.Duration // Token expiration duration (default: 24 hours)
}

// Claims represents the JWT token payload
type Claims struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Roles     []string  `json:"roles,omitempty"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

// New creates a new JWT session manager
func New(config Config) *JWTSessionManager {
	expiration := config.Expiration
	if expiration == 0 {
		expiration = 24 * time.Hour // Default to 24 hours
	}

	return &JWTSessionManager{
		secret:     []byte(config.Secret),
		expiration: expiration,
		log:        logger.NewServiceLogger("jwt-session"),
	}
}

// CreateSession generates a new JWT token for the given user ID
func (j *JWTSessionManager) CreateSession(userId string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    userId,
		IssuedAt:  now,
		ExpiresAt: now.Add(j.expiration),
	}

	token, err := j.generateToken(claims)
	if err != nil {
		j.log.Error("failed to generate token", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userId,
		})
		return "", err
	}

	j.log.Info("session created", map[string]interface{}{
		"user_id":    userId,
		"expires_at": claims.ExpiresAt,
	})

	return token, nil
}

// CreateSessionWithUser generates a new JWT token with user details
func (j *JWTSessionManager) CreateSessionWithUser(user *core.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:    user.ID,
		Email:     user.Email,
		Roles:     user.Roles,
		IssuedAt:  now,
		ExpiresAt: now.Add(j.expiration),
	}

	token, err := j.generateToken(claims)
	if err != nil {
		j.log.Error("failed to generate token with user", map[string]interface{}{
			"error":   err.Error(),
			"user_id": user.ID,
			"email":   user.Email,
		})
		return "", err
	}

	j.log.Info("session created with user details", map[string]interface{}{
		"user_id":    user.ID,
		"email":      user.Email,
		"expires_at": claims.ExpiresAt,
	})

	return token, nil
}

// ValidateSession validates a JWT token and returns the user ID
func (j *JWTSessionManager) ValidateSession(ctx *core.Context, token string) (string, error) {
	claims, err := j.parseToken(token)
	if err != nil {
		j.log.Warn("token validation failed", map[string]interface{}{
			"error":      err.Error(),
			"request_id": ctx.RequestID,
		})
		return "", err
	}

	// Check expiration
	if time.Now().After(claims.ExpiresAt) {
		j.log.Warn("token expired", map[string]interface{}{
			"user_id":    claims.UserID,
			"expired_at": claims.ExpiresAt,
			"request_id": ctx.RequestID,
		})
		return "", ErrExpiredToken
	}

	j.log.Debug("token validated successfully", map[string]interface{}{
		"user_id":    claims.UserID,
		"request_id": ctx.RequestID,
	})

	return claims.UserID, nil
}

// ValidateSessionWithClaims validates a JWT token and returns the full claims
func (j *JWTSessionManager) ValidateSessionWithClaims(ctx *core.Context, token string) (*Claims, error) {
	claims, err := j.parseToken(token)
	if err != nil {
		j.log.Warn("token validation failed", map[string]interface{}{
			"error":      err.Error(),
			"request_id": ctx.RequestID,
		})
		return nil, err
	}

	// Check expiration
	if time.Now().After(claims.ExpiresAt) {
		j.log.Warn("token expired", map[string]interface{}{
			"user_id":    claims.UserID,
			"expired_at": claims.ExpiresAt,
			"request_id": ctx.RequestID,
		})
		return nil, ErrExpiredToken
	}

	j.log.Debug("token validated with claims", map[string]interface{}{
		"user_id":    claims.UserID,
		"email":      claims.Email,
		"request_id": ctx.RequestID,
	})

	return claims, nil
}

// DestroySession destroys a session (JWT tokens are stateless, so this is a no-op)
// In a production system, you might maintain a blacklist of revoked tokens
func (j *JWTSessionManager) DestroySession(ctx *core.Context, token string) error {
	// Parse token to get user ID for logging
	claims, err := j.parseToken(token)
	if err != nil {
		j.log.Warn("failed to destroy session: invalid token", map[string]interface{}{
			"error":      err.Error(),
			"request_id": ctx.RequestID,
		})
		return err
	}

	j.log.Info("session destroyed", map[string]interface{}{
		"user_id":    claims.UserID,
		"request_id": ctx.RequestID,
	})

	// Note: In a real implementation, you would add the token to a blacklist
	// or use token versioning in the database
	return nil
}

// generateToken creates a JWT token from claims
func (j *JWTSessionManager) generateToken(claims Claims) (string, error) {
	// Create header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", err
	}
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerJSON)

	// Create payload
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadJSON)

	// Create signature
	message := headerEncoded + "." + payloadEncoded
	signature := j.sign(message)
	signatureEncoded := base64.RawURLEncoding.EncodeToString(signature)

	// Combine all parts
	token := message + "." + signatureEncoded

	return token, nil
}

// parseToken parses and validates a JWT token
func (j *JWTSessionManager) parseToken(token string) (*Claims, error) {
	// Split token into parts
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	headerEncoded := parts[0]
	payloadEncoded := parts[1]
	signatureEncoded := parts[2]

	// Verify signature
	message := headerEncoded + "." + payloadEncoded
	expectedSignature := j.sign(message)
	expectedSignatureEncoded := base64.RawURLEncoding.EncodeToString(expectedSignature)

	if signatureEncoded != expectedSignatureEncoded {
		return nil, ErrInvalidSignature
	}

	// Decode payload
	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadEncoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	// Unmarshal claims
	var claims Claims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	return &claims, nil
}

// sign creates an HMAC-SHA256 signature
func (j *JWTSessionManager) sign(message string) []byte {
	h := hmac.New(sha256.New, j.secret)
	h.Write([]byte(message))
	return h.Sum(nil)
}
