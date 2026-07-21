package errorhandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackman0925/gin-middleware/response"
)

func TestErrorHandler_CatchesError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/error", func(c *gin.Context) {
		c.Error(errors.New("something went wrong"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/error", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	var resp response.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected code 500, got %d", resp.Code)
	}
	if resp.Message != defaultErrorMessage {
		t.Fatalf("expected private error to be hidden, got %s", resp.Message)
	}
}

func TestErrorHandler_ExposesExplicitPublicError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/error", func(c *gin.Context) {
		c.Error(errors.New("safe client message")).SetType(gin.ErrorTypePublic)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/error", nil)
	r.ServeHTTP(w, req)

	var resp response.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Message != "safe client message" {
		t.Fatalf("expected public message, got %q", resp.Message)
	}
}

func TestErrorHandler_CustomMapper(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandlerWithMapper(func(err *gin.Error) (int, string) {
		if errors.Is(err.Err, errNotFound) {
			return http.StatusNotFound, "resource not found"
		}
		return http.StatusInternalServerError, ""
	}))
	r.GET("/error", func(c *gin.Context) { c.Error(errNotFound) })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/error", nil)
	r.ServeHTTP(w, req)

	var resp response.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusNotFound || resp.Message != "resource not found" {
		t.Fatalf("unexpected mapped response: status=%d response=%+v", w.Code, resp)
	}
}

var errNotFound = errors.New("not found")

func TestErrorHandler_DoesNotOverrideWrittenResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/partial", func(c *gin.Context) {
		c.String(http.StatusBadRequest, "partial response")
		c.Error(errors.New("ignored error"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/partial", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}
	if w.Body.String() != "partial response" {
		t.Fatalf("expected body 'partial response', got %s", w.Body.String())
	}
}

func TestErrorHandler_NoError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ErrorHandler())
	r.GET("/ok", func(c *gin.Context) {
		response.Success(c, gin.H{"key": "value"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ok", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp response.APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}
	if resp.Message != "success" {
		t.Fatalf("expected message 'success', got %s", resp.Message)
	}
}
