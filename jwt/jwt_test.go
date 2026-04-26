package jwt

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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

	token, err := j.GenerateTokenWithUsername("admin", map[string]interface{}{
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
		c.JSON(200, gin.H{"username": username})
	})

	// Valid token
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	// Missing header
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	// Invalid prefix
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Token "+token)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}
