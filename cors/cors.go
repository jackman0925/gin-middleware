// Package cors provides CORS middleware for Gin.
package cors

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
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

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allow := false
		isWildcard := false
		for _, o := range config.AllowedOrigins {
			if o == "*" {
				isWildcard = true
				allow = true
				break
			}
			if o == origin {
				allow = true
				break
			}
		}
		if len(config.AllowedOrigins) == 0 {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else if isWildcard {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else if allow {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}

		c.Writer.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAge))
		c.Writer.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))
		c.Writer.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))
		if config.AllowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if c.Request.Method == "OPTIONS" {
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
