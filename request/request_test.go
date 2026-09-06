package request

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type bindRequest struct {
	Name     string        `json:"name" binding:"required"`
	Page     int           `json:"page" default:"1" binding:"required,min=1"`
	Enabled  bool          `json:"enabled" default:"true"`
	Timeout  time.Duration `json:"timeout" default:"5s"`
	Tags     []string      `json:"tags" default:"[\"general\"]"`
	Retries  *int          `json:"retries" default:"3"`
	Ignored  string        `json:"ignored" default:"-"`
	Settings bindSettings  `json:"settings"`
}

type bindSettings struct {
	Limit int `json:"limit" default:"20" binding:"min=1,max=100"`
}

func TestBindJSONAppliesDefaultsBeforeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	c.Request.Body = ioNopCloser(`{"name":"Ada"}`)

	var request bindRequest
	if err := BindJSON(c, &request); err != nil {
		t.Fatalf("BindJSON returned an error: %v", err)
	}
	if request.Page != 1 || !request.Enabled || request.Timeout != 5*time.Second || request.Settings.Limit != 20 {
		t.Fatalf("defaults were not applied: %+v", request)
	}
	if request.Retries == nil || *request.Retries != 3 {
		t.Fatalf("pointer default was not applied: %+v", request.Retries)
	}
	if request.Ignored != "" {
		t.Fatalf("default tag - should be ignored, got %q", request.Ignored)
	}
	if len(request.Tags) != 1 || request.Tags[0] != "general" {
		t.Fatalf("slice default was not applied: %+v", request.Tags)
	}
}

func TestBindJSONPreservesExplicitZeroValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	c.Request.Body = ioNopCloser(`{"name":"Ada","enabled":false,"timeout":0,"settings":{"limit":0}}`)

	var request bindRequest
	err := BindJSON(c, &request)
	if err == nil {
		t.Fatal("explicit nested limit 0 should fail validation, not be overwritten by a default")
	}
	if request.Enabled || request.Timeout != 0 || request.Settings.Limit != 0 {
		t.Fatalf("explicit JSON values were overwritten: %+v", request)
	}
}

func TestBindJSONCachesBodyForAnotherBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	c.Request.Body = ioNopCloser(`{"name":"Ada"}`)

	var first bindRequest
	if err := BindJSON(c, &first); err != nil {
		t.Fatal(err)
	}
	var second struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindBodyWithJSON(&second); err != nil {
		t.Fatalf("cached body could not be reused: %v", err)
	}
	if second.Name != "Ada" {
		t.Fatalf("unexpected second binding: %+v", second)
	}
}

func TestSetReqDefaults(t *testing.T) {
	value := bindRequest{}
	if err := SetReqDefaults(&value); err != nil {
		t.Fatal(err)
	}
	if value.Page != 1 || !value.Enabled || value.Settings.Limit != 20 {
		t.Fatalf("defaults were not applied: %+v", value)
	}
}

func TestSetReqDefaultsRejectsInvalidDefaultAndTarget(t *testing.T) {
	type invalidDefault struct {
		Count int `default:"not-a-number"`
	}
	if err := SetReqDefaults(&invalidDefault{}); err == nil {
		t.Fatal("expected invalid default error")
	}
	if err := SetReqDefaults(bindRequest{}); !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("expected ErrInvalidTarget, got %v", err)
	}
}

func TestBindJSONReportsValidationErrorsAfterDefaults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", http.NoBody)
	c.Request.Body = ioNopCloser(`{}`)

	var request bindRequest
	if err := BindJSON(c, &request); err == nil {
		t.Fatal("expected required name validation error")
	}
}

func ioNopCloser(body string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(body))
}
