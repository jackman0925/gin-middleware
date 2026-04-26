// Package jwt provides JWT authentication middleware for Gin.
package jwt

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackman0925/gin-middleware/response"
)

type contextKey string

const (
	claimsKey   contextKey = "_jwt_claims"
	usernameKey contextKey = "username"
)

// Config holds the JWT configuration
type Config struct {
	// Secret is the signing key for JWT
	Secret string
	// TokenHeaderName is the header name for JWT token (default: "Authorization")
	TokenHeaderName string
	// TokenPrefix is the prefix for JWT token (default: "Bearer")
	TokenPrefix string
	// Expiration is the token expiration duration (default: 72 hours)
	Expiration time.Duration
	// SigningMethod is the JWT signing method (default: HS256)
	SigningMethod jwt.SigningMethod
}

// DefaultConfig returns a Config with default values
func DefaultConfig(secret string) Config {
	return Config{
		Secret:          secret,
		TokenHeaderName: "Authorization",
		TokenPrefix:     "Bearer",
		Expiration:      time.Hour * 72,
		SigningMethod:   jwt.SigningMethodHS256,
	}
}

// JWT is the middleware instance
type JWT struct {
	Config Config
}

// Claims represents the JWT claims
type Claims jwt.MapClaims

// New creates a new JWT middleware with the given secret
func New(secret string) *JWT {
	return &JWT{Config: DefaultConfig(secret)}
}

// NewWithConfig creates a new JWT middleware with custom configuration
func NewWithConfig(config Config) *JWT {
	return &JWT{Config: config}
}

// Validate checks if the configuration is valid
func (j *JWT) Validate() error {
	if j.Config.Secret == "" {
		return errors.New("jwt secret is required")
	}
	if len(j.Config.Secret) < 32 {
		return errors.New("jwt secret should be at least 32 characters for security")
	}
	return nil
}

// GenerateToken creates a new JWT token with the given claims
func (j *JWT) GenerateToken(claims jwt.MapClaims) (string, error) {
	if err := j.Validate(); err != nil {
		return "", err
	}

	// Set default expiration if not provided
	if _, exists := claims["exp"]; !exists {
		claims["exp"] = time.Now().Add(j.Config.Expiration).Unix()
	}

	token := jwt.NewWithClaims(j.Config.SigningMethod, claims)
	tokenString, err := token.SignedString([]byte(j.Config.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// GenerateTokenWithUsername creates a JWT token for a user with username and optional metadata
func (j *JWT) GenerateTokenWithUsername(username string, metadata map[string]any) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"iat":      time.Now().Unix(),
	}

	// Add additional metadata
	for k, v := range metadata {
		claims[k] = v
	}

	return j.GenerateToken(claims)
}

// ParseToken parses and validates a JWT token
func (j *JWT) ParseToken(tokenString string) (jwt.MapClaims, error) {
	if err := j.Validate(); err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.Config.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid or expired token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// Middleware returns the Gin middleware handler
func (j *JWT) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(j.Config.TokenHeaderName)
		if authHeader == "" {
			response.FailWithMessage(c, http.StatusUnauthorized, "authorization header is missing")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != j.Config.TokenPrefix {
			response.FailWithMessage(c, http.StatusUnauthorized, fmt.Sprintf("authorization header format must be %s {token}", j.Config.TokenPrefix))
			return
		}

		tokenString := parts[1]
		claims, err := j.ParseToken(tokenString)
		if err != nil {
			response.FailWithMessage(c, http.StatusUnauthorized, err.Error())
			return
		}

		// Set claims in context using private keys
		c.Set(string(claimsKey), claims)
		for k, v := range claims {
			// For backward compatibility and ease of use, we still set string keys
			// but internal helpers will prefer the typed key if possible.
			c.Set(k, v)
		}

		c.Next()
	}
}

// ClaimsFromContext retrieves JWT claims from the Gin context
func ClaimsFromContext(c *gin.Context) (jwt.MapClaims, bool) {
	claims, exists := c.Get(string(claimsKey))
	if !exists {
		return nil, false
	}
	return claims.(jwt.MapClaims), true
}

// UsernameFromContext retrieves the username from JWT claims in context
func UsernameFromContext(c *gin.Context) (string, bool) {
	username, exists := c.Get(string(usernameKey))
	if !exists {
		// Fallback to string key if private key not found
		username, exists = c.Get("username")
		if !exists {
			return "", false
		}
	}
	val, ok := username.(string)
	return val, ok
}

