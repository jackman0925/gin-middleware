package cors

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSAllowAll(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AllowAll())
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected Allow-Origin *, got %s", w.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSWithOrigins(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(New([]string{"https://allowed.com"}))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// 1. Allowed origin
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://allowed.com")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Vary") != "Origin" {
		t.Fatal("expected Vary: Origin header for dynamic origin")
	}

	// 2. Disallowed origin - should use unified response format
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://blocked.com")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "\"code\":403") {
		t.Fatalf("response should match unified fail format, got: %s", w.Body.String())
	}
}

func TestCORSPreflight(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AllowAll())
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", w.Code)
	}
}

func TestCORSWildcardDefaultsToNoCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(New([]string{"*"}))
	r.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("wildcard origin must not enable credentials, got %q", got)
	}
}

func TestCORSWildcardWithCredentialsReflectsOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewWithConfig(Config{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
	}))
	r.GET("/test", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	r.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("expected reflected origin, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials header, got %q", got)
	}
	if got := w.Header().Values("Vary"); !containsHeaderValue(got, "Origin") {
		t.Fatalf("expected Vary: Origin, got %v", got)
	}
}

func TestCORSRejectedOriginStillVariesByOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(New([]string{"https://allowed.example"}))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://blocked.example")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	if got := w.Header().Values("Vary"); !containsHeaderValue(got, "Origin") {
		t.Fatalf("expected rejected response to vary by Origin, got %v", got)
	}
}

func TestCORSPreflightRejectsMethodAndHeaders(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		headers string
		message string
	}{
		{"method", http.MethodDelete, "Content-Type", "CORS method not allowed"},
		{"header", http.MethodGet, "X-Forbidden", "CORS header not allowed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.Use(NewWithConfig(Config{
				AllowedOrigins: []string{"https://example.com"},
				AllowedMethods: []string{http.MethodGet},
				AllowedHeaders: []string{"Content-Type"},
			}))

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodOptions, "/test", nil)
			req.Header.Set("Origin", "https://example.com")
			req.Header.Set("Access-Control-Request-Method", tt.method)
			req.Header.Set("Access-Control-Request-Headers", tt.headers)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), tt.message) {
				t.Fatalf("expected 403 %q, got status=%d body=%s", tt.message, w.Code, w.Body.String())
			}
		})
	}
}

func containsHeaderValue(values []string, expected string) bool {
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(part), expected) {
				return true
			}
		}
	}
	return false
}
