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
	allowCredentials := true
	for _, origin := range origins {
		if origin == "*" {
			// Browsers reject Allow-Origin "*" together with credentials.
			allowCredentials = false
			break
		}
	}
	return NewWithConfig(Config{
		AllowedOrigins:   origins,
		AllowedMethods:   DefaultConfig().AllowedMethods,
		AllowedHeaders:   DefaultConfig().AllowedHeaders,
		AllowCredentials: allowCredentials,
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
	allowedMethodsMap := make(map[string]struct{}, len(methods))
	allowAnyMethod := false
	for _, method := range methods {
		method = strings.ToUpper(strings.TrimSpace(method))
		if method == "*" {
			allowAnyMethod = true
		}
		allowedMethodsMap[method] = struct{}{}
	}
	allowedHeadersMap := make(map[string]struct{}, len(headers))
	allowAnyHeader := false
	for _, header := range headers {
		header = strings.ToLower(strings.TrimSpace(header))
		if header == "*" {
			allowAnyHeader = true
		}
		allowedHeadersMap[header] = struct{}{}
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

		// A wildcard cannot be sent with credentials, so credentialed wildcard
		// configurations reflect the concrete origin and vary the response.
		dynamicOrigin := !hasWildcard || config.AllowCredentials
		if dynamicOrigin {
			c.Writer.Header().Add("Vary", "Origin")
		}

		allow := false
		if len(config.AllowedOrigins) == 0 {
			allow = true
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else if hasWildcard {
			allow = true
			if config.AllowCredentials {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			}
		} else if _, ok := allowedOriginsMap[origin]; ok {
			allow = true
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		}

		if !allow {
			log.Warnf("CORS origin blocked: %s for %s %s", origin, c.Request.Method, c.Request.URL.Path)
			response.FailWithMessage(c, http.StatusForbidden, "CORS origin not allowed")
			return
		}

		if c.Request.Method == http.MethodOptions {
			c.Writer.Header().Add("Vary", "Access-Control-Request-Method")
			c.Writer.Header().Add("Vary", "Access-Control-Request-Headers")

			requestedMethod := strings.ToUpper(strings.TrimSpace(c.GetHeader("Access-Control-Request-Method")))
			if requestedMethod != "" && !allowAnyMethod {
				if _, ok := allowedMethodsMap[requestedMethod]; !ok {
					log.Warnf("CORS method blocked: %s for origin %s", requestedMethod, origin)
					response.FailWithMessage(c, http.StatusForbidden, "CORS method not allowed")
					return
				}
			}

			if requestedHeaders := c.GetHeader("Access-Control-Request-Headers"); requestedHeaders != "" && !allowAnyHeader {
				for _, requestedHeader := range strings.Split(requestedHeaders, ",") {
					requestedHeader = strings.ToLower(strings.TrimSpace(requestedHeader))
					if _, ok := allowedHeadersMap[requestedHeader]; !ok {
						log.Warnf("CORS header blocked: %s for origin %s", requestedHeader, origin)
						response.FailWithMessage(c, http.StatusForbidden, "CORS header not allowed")
						return
					}
				}
			}
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
