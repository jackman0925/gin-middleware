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
	"github.com/jackman0925/gin-middleware/log"
	"github.com/jackman0925/gin-middleware/response"
)

type contextKey string

const (
	claimsKey   contextKey = "github.com/jackman0925/gin-middleware/jwt.claims"
	usernameKey contextKey = "username"
)

// ErrInvalidToken is returned when a token cannot be parsed or validated.
var ErrInvalidToken = errors.New("invalid or expired token")

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
	// Issuer, when set, is required to match the token's iss claim.
	Issuer string
	// Audience, when set, is required to be present in the token's aud claim.
	Audience string
	// Leeway allows a small amount of clock skew when validating time claims.
	Leeway time.Duration
	// AllowMissingExpiration permits tokens without an exp claim. It is false by
	// default so tokens are required to expire.
	AllowMissingExpiration bool
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
	return &JWT{Config: withDefaults(config)}
}

func withDefaults(config Config) Config {
	defaults := DefaultConfig(config.Secret)
	if config.TokenHeaderName == "" {
		config.TokenHeaderName = defaults.TokenHeaderName
	}
	if config.TokenPrefix == "" {
		config.TokenPrefix = defaults.TokenPrefix
	}
	if config.Expiration == 0 {
		config.Expiration = defaults.Expiration
	}
	if config.SigningMethod == nil {
		config.SigningMethod = defaults.SigningMethod
	}
	return config
}

// Validate checks if the configuration is valid
func (j *JWT) Validate() error {
	if j == nil {
		return errors.New("jwt configuration is required")
	}
	if j.Config.Secret == "" {
		return errors.New("jwt secret is required")
	}
	if len(j.Config.Secret) < 32 {
		return errors.New("jwt secret should be at least 32 characters for security")
	}
	if j.Config.TokenHeaderName == "" {
		return errors.New("jwt token header name is required")
	}
	if j.Config.TokenPrefix == "" {
		return errors.New("jwt token prefix is required")
	}
	if j.Config.Expiration <= 0 {
		return errors.New("jwt expiration must be greater than zero")
	}
	if j.Config.Leeway < 0 {
		return errors.New("jwt leeway cannot be negative")
	}
	if j.Config.SigningMethod == nil {
		return errors.New("jwt signing method is required")
	}
	if _, ok := j.Config.SigningMethod.(*jwt.SigningMethodHMAC); !ok {
		return fmt.Errorf("unsupported jwt signing method %q: only HMAC methods are supported", j.Config.SigningMethod.Alg())
	}
	return nil
}

// GenerateToken creates a new JWT token with the given claims
func (j *JWT) GenerateToken(claims jwt.MapClaims) (string, error) {
	if err := j.Validate(); err != nil {
		return "", err
	}

	// Copy the map so token generation never mutates caller-owned claims.
	tokenClaims := make(jwt.MapClaims, len(claims)+1)
	for key, value := range claims {
		tokenClaims[key] = value
	}

	if _, exists := tokenClaims["exp"]; !exists {
		tokenClaims["exp"] = time.Now().Add(j.Config.Expiration).Unix()
	}

	token := jwt.NewWithClaims(j.Config.SigningMethod, tokenClaims)
	tokenString, err := token.SignedString([]byte(j.Config.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// GenerateTokenWithUsername creates a JWT token for a user with username and optional metadata
func (j *JWT) GenerateTokenWithUsername(username string, metadata map[string]any) (string, error) {
	claims := make(jwt.MapClaims, len(metadata)+2)

	// Add metadata first so callers cannot replace identity and issued-at claims.
	for k, v := range metadata {
		claims[k] = v
	}
	claims["username"] = username
	claims["iat"] = time.Now().Unix()

	return j.GenerateToken(claims)
}

// ParseToken parses and validates a JWT token
func (j *JWT) ParseToken(tokenString string) (jwt.MapClaims, error) {
	if err := j.Validate(); err != nil {
		return nil, err
	}

	options := []jwt.ParserOption{
		jwt.WithValidMethods([]string{j.Config.SigningMethod.Alg()}),
	}
	if !j.Config.AllowMissingExpiration {
		options = append(options, jwt.WithExpirationRequired())
	}
	if j.Config.Leeway > 0 {
		options = append(options, jwt.WithLeeway(j.Config.Leeway))
	}
	if j.Config.Issuer != "" {
		options = append(options, jwt.WithIssuer(j.Config.Issuer))
	}
	if j.Config.Audience != "" {
		options = append(options, jwt.WithAudience(j.Config.Audience))
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != j.Config.SigningMethod.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.Config.Secret), nil
	}, options...)

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// Middleware returns the Gin middleware handler
func (j *JWT) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := j.Validate(); err != nil {
			log.Errorf("invalid JWT middleware configuration: %v", err)
			response.FailWithMessage(c, http.StatusInternalServerError, "authentication service unavailable")
			return
		}

		authHeader := c.GetHeader(j.Config.TokenHeaderName)
		if authHeader == "" {
			log.Warnf("missing authorization header from %s %s", c.Request.Method, c.Request.URL.Path)
			response.FailWithMessage(c, http.StatusUnauthorized, "authorization header is missing")
			return
		}

		parts := strings.Fields(authHeader)
		if len(parts) != 2 || !strings.EqualFold(parts[0], j.Config.TokenPrefix) {
			log.Warnf("invalid authorization header format from %s %s", c.Request.Method, c.Request.URL.Path)
			response.FailWithMessage(c, http.StatusUnauthorized, fmt.Sprintf("authorization header format must be %s {token}", j.Config.TokenPrefix))
			return
		}

		tokenString := parts[1]
		claims, err := j.ParseToken(tokenString)
		if err != nil {
			log.Warnf("invalid token from %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
			response.FailWithMessage(c, http.StatusUnauthorized, ErrInvalidToken.Error())
			return
		}

		// Store claims under a package-owned namespace. Flattening arbitrary claims
		// into Gin context could overwrite values set by other middleware.
		c.Set(string(claimsKey), claims)

		c.Next()
	}
}

// ClaimsFromContext retrieves JWT claims from the Gin context
func ClaimsFromContext(c *gin.Context) (jwt.MapClaims, bool) {
	claims, exists := c.Get(string(claimsKey))
	if !exists {
		return nil, false
	}
	value, ok := claims.(jwt.MapClaims)
	return value, ok
}

// UsernameFromContext retrieves the username from JWT claims in context
func UsernameFromContext(c *gin.Context) (string, bool) {
	if claims, ok := ClaimsFromContext(c); ok {
		username, ok := claims["username"].(string)
		return username, ok
	}

	// Keep compatibility with applications that populated username directly.
	username, exists := c.Get(string(usernameKey))
	if !exists {
		return "", false
	}
	val, ok := username.(string)
	return val, ok
}
