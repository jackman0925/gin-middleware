package jwt

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	jwtlib "github.com/golang-jwt/jwt/v5"
)

func TestGenerateToken(t *testing.T) {
	j := New("test-secret-key-that-is-at-least-32-chars")

	token, err := j.GenerateTokenWithUsername("admin", map[string]interface{}{
		"adminID": 1,
	})
	if err != nil {
		t.Fatalf("GenerateTokenWithUsername failed: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}
}

func TestNewWithConfigAppliesDefaults(t *testing.T) {
	j := NewWithConfig(Config{Secret: "test-secret-key-that-is-at-least-32-chars"})
	if j.Config.TokenHeaderName != "Authorization" || j.Config.TokenPrefix != "Bearer" {
		t.Fatalf("expected default header configuration, got %#v", j.Config)
	}
	if j.Config.Expiration != 72*time.Hour || j.Config.SigningMethod != jwtlib.SigningMethodHS256 {
		t.Fatalf("expected default token configuration, got %#v", j.Config)
	}
	if err := j.Validate(); err != nil {
		t.Fatalf("defaulted configuration should be valid: %v", err)
	}
}

func TestValidateRejectsUnsupportedConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*JWT)
	}{
		{"missing header", func(j *JWT) { j.Config.TokenHeaderName = "" }},
		{"missing prefix", func(j *JWT) { j.Config.TokenPrefix = "" }},
		{"negative expiration", func(j *JWT) { j.Config.Expiration = -time.Second }},
		{"negative leeway", func(j *JWT) { j.Config.Leeway = -time.Second }},
		{"missing method", func(j *JWT) { j.Config.SigningMethod = nil }},
		{"non HMAC method", func(j *JWT) { j.Config.SigningMethod = jwtlib.SigningMethodRS256 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := New("test-secret-key-that-is-at-least-32-chars")
			tt.mutate(j)
			if err := j.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestGenerateTokenDoesNotMutateClaims(t *testing.T) {
	j := New("test-secret-key-that-is-at-least-32-chars")
	claims := jwtlib.MapClaims{"subject": "123"}
	if _, err := j.GenerateToken(claims); err != nil {
		t.Fatal(err)
	}
	if _, exists := claims["exp"]; exists {
		t.Fatal("GenerateToken must not add exp to the caller's map")
	}
}

func TestMetadataCannotOverrideIdentityClaims(t *testing.T) {
	j := New("test-secret-key-that-is-at-least-32-chars")
	token, err := j.GenerateTokenWithUsername("expected", map[string]any{
		"username": "attacker",
		"iat":      int64(1),
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := j.ParseToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims["username"] != "expected" {
		t.Fatalf("username was overwritten: %v", claims["username"])
	}
	if claims["iat"].(float64) == 1 {
		t.Fatal("iat was overwritten by metadata")
	}
}

func TestParseTokenRejectsDifferentHMACAlgorithm(t *testing.T) {
	secret := "test-secret-key-that-is-at-least-32-chars"
	j := NewWithConfig(Config{Secret: secret, SigningMethod: jwtlib.SigningMethodHS512})
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.ParseToken(signed); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for algorithm mismatch, got %v", err)
	}
}

func TestParseTokenRequiresExpirationByDefault(t *testing.T) {
	secret := "test-secret-key-that-is-at-least-32-chars"
	j := New(secret)
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, jwtlib.MapClaims{"sub": "123"})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.ParseToken(signed); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected missing expiration to be rejected, got %v", err)
	}

	j.Config.AllowMissingExpiration = true
	if _, err := j.ParseToken(signed); err != nil {
		t.Fatalf("expected missing expiration to be allowed explicitly: %v", err)
	}
}

func TestParseTokenValidatesIssuerAudienceAndLeeway(t *testing.T) {
	secret := "test-secret-key-that-is-at-least-32-chars"
	j := NewWithConfig(Config{
		Secret:   secret,
		Issuer:   "expected-issuer",
		Audience: "expected-audience",
		Leeway:   time.Minute,
	})

	valid, err := j.GenerateToken(jwtlib.MapClaims{
		"iss": "expected-issuer",
		"aud": "expected-audience",
		"exp": time.Now().Add(-30 * time.Second).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.ParseToken(valid); err != nil {
		t.Fatalf("expected configured leeway to accept token: %v", err)
	}

	wrongIssuer, err := j.GenerateToken(jwtlib.MapClaims{
		"iss": "wrong-issuer",
		"aud": "expected-audience",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := j.ParseToken(wrongIssuer); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected issuer validation error, got %v", err)
	}
}

func TestParseToken(t *testing.T) {
	j := New("test-secret-key-that-is-at-least-32-chars")

	token, err := j.GenerateTokenWithUsername("admin", map[string]interface{}{
		"adminID": 1,
	})
	if err != nil {
		t.Fatalf("GenerateTokenWithUsername failed: %v", err)
	}

	claims, err := j.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken failed: %v", err)
	}
	if claims["username"] != "admin" {
		t.Fatalf("expected username 'admin', got %v", claims["username"])
	}
	if claims["adminID"].(float64) != 1 {
		t.Fatalf("expected adminID 1, got %v", claims["adminID"])
	}
}

func TestParseTokenInvalid(t *testing.T) {
	j := New("test-secret-key-that-is-at-least-32-chars")

	_, err := j.ParseToken("invalid.token.here")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
}

func TestParseTokenWrongSecret(t *testing.T) {
	j1 := New("test-secret-key-that-is-at-least-32-chars")
	j2 := New("different-secret-key-that-is-at-least-32-chars")

	token, err := j1.GenerateTokenWithUsername("admin", nil)
	if err != nil {
		t.Fatalf("GenerateTokenWithUsername failed: %v", err)
	}

	_, err = j2.ParseToken(token)
	if err == nil {
		t.Fatal("expected error when parsing with different secret")
	}
}

func TestValidate(t *testing.T) {
	j := New("")
	if err := j.Validate(); err == nil {
		t.Fatal("expected error for empty secret")
	}

	j = New("short")
	if err := j.Validate(); err == nil {
		t.Fatal("expected error for short secret")
	}

	j = New("test-secret-key-that-is-at-least-32-chars")
	if err := j.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMiddleware(t *testing.T) {
	j := New("test-secret-key-that-is-at-least-32-chars")

	token, err := j.GenerateTokenWithUsername("admin", map[string]any{
		"adminID": 1,
	})
	if err != nil {
		t.Fatalf("GenerateTokenWithUsername failed: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(j.Middleware())
	r.GET("/test", func(c *gin.Context) {
		username, _ := UsernameFromContext(c)
		// Also verify we can get claims
		claims, _ := ClaimsFromContext(c)
		c.JSON(200, gin.H{
			"username": username,
			"adminID":  claims["adminID"],
		})
	})

	// 1. Valid token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Claims are namespaced and must not overwrite unrelated context values.
	r = gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("adminID", "application-value")
	})
	r.Use(j.Middleware())
	r.GET("/context", func(c *gin.Context) {
		value, _ := c.Get("adminID")
		c.String(http.StatusOK, "%v", value)
	})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/context", nil)
	req.Header.Set("Authorization", "bearer   "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "application-value" {
		t.Fatalf("claims should not overwrite context; status=%d body=%q", w.Code, w.Body.String())
	}

	// 2. Missing header - should use unified response format
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "\"code\":401") || !strings.Contains(w.Body.String(), "\"message\"") {
		t.Fatalf("response should match unified fail format, got: %s", w.Body.String())
	}

	// 3. Invalid prefix
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Token "+token)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}

func TestMiddlewareDoesNotExposeParserDetails(t *testing.T) {
	j := New("test-secret-key-that-is-at-least-32-chars")
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(j.Middleware())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.value")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), ErrInvalidToken.Error()) || strings.Contains(w.Body.String(), "token is malformed") {
		t.Fatalf("response leaked parser details: %s", w.Body.String())
	}
}

func TestMiddlewareRejectsInvalidConfigurationSafely(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var j *JWT
	r := gin.New()
	r.Use(j.Middleware())
	r.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "authentication service unavailable") {
		t.Fatalf("expected safe configuration failure, status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestClaimsFromContextHandlesWrongType(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set(string(claimsKey), "not claims")
	if _, ok := ClaimsFromContext(c); ok {
		t.Fatal("wrong context value type should be rejected")
	}
}
