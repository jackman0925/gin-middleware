// Package cors provides CORS middleware for Gin.
package cors

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackman0925/gin-middleware/log"
	"github.com/jackman0925/gin-middleware/response"
)

// Config holds the CORS configuration
type Config struct {
	// AllowedOrigins is the list of allowed origins. Empty means allow all.
	AllowedOrigins []string
	// AllowedMethods is the list of allowed HTTP methods (default: common methods)
	AllowedMethods []string
	// AllowedHeaders is the list of allowed headers (default: common headers)
	AllowedHeaders []string
	// AllowCredentials allows credentials to be sent (default: true)
	AllowCredentials bool
	// MaxAge sets Access-Control-Max-Age in seconds (default: 86400)
	MaxAge int
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Accept", "Origin", "Cache-Control", "X-Requested-With"},
		AllowCredentials: true,
		MaxAge:           86400,
	}
}

// New creates a CORS middleware with the given allowed origins
func New(origins []string) gin.HandlerFunc {
	return NewWithConfig(Config{
		AllowedOrigins:   origins,
		AllowedMethods:   DefaultConfig().AllowedMethods,
		AllowedHeaders:   DefaultConfig().AllowedHeaders,
		AllowCredentials: true,
		MaxAge:           86400,
	})
}

// NewWithConfig creates a CORS middleware with custom configuration
func NewWithConfig(config Config) gin.HandlerFunc {
	methods := config.AllowedMethods
	if len(methods) == 0 {
		methods = DefaultConfig().AllowedMethods
	}
	headers := config.AllowedHeaders
	if len(headers) == 0 {
		headers = DefaultConfig().AllowedHeaders
	}
	maxAge := config.MaxAge
	if maxAge == 0 {
		maxAge = 86400
	}

	// Pre-process origins for O(1) lookup
	allowedOriginsMap := make(map[string]struct{}, len(config.AllowedOrigins))
	hasWildcard := false
	for _, o := range config.AllowedOrigins {
		if o == "*" {
			hasWildcard = true
		}
		allowedOriginsMap[o] = struct{}{}
	}

	allowedMethodsStr := strings.Join(methods, ", ")
	allowedHeadersStr := strings.Join(headers, ", ")
	maxAgeStr := strconv.Itoa(maxAge)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		allow := false
		if len(config.AllowedOrigins) == 0 {
			allow = true
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else if hasWildcard {
			allow = true
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else if _, ok := allowedOriginsMap[origin]; ok {
			allow = true
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		if !allow {
			log.Warnf("CORS origin blocked: %s for %s %s", origin, c.Request.Method, c.Request.URL.Path)
			response.FailWithMessage(c, http.StatusForbidden, "CORS origin not allowed")
			return
		}

		// When Access-Control-Allow-Origin is dynamic, we must set Vary: Origin
		if !hasWildcard {
			c.Writer.Header().Add("Vary", "Origin")
		}

		c.Writer.Header().Set("Access-Control-Max-Age", maxAgeStr)
		c.Writer.Header().Set("Access-Control-Allow-Methods", allowedMethodsStr)
		c.Writer.Header().Set("Access-Control-Allow-Headers", allowedHeadersStr)
		if config.AllowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// AllowAll creates a CORS middleware that allows everything (development use)
func AllowAll() gin.HandlerFunc {
	return NewWithConfig(Config{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   DefaultConfig().AllowedMethods,
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		MaxAge:           86400,
	})
}
