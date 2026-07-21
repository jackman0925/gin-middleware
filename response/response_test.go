package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		Success(c, gin.H{"key": "value"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp APIResponse
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

func TestSuccessWithCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		SuccessWithCode(c, 1001, "custom success", gin.H{"key": "value"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	var resp APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusOK || resp.Code != 1001 || resp.Message != "custom success" {
		t.Fatalf("unexpected response: status=%d response=%+v", w.Code, resp)
	}
}

func TestFail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		Fail(c, http.StatusNotFound, errors.New("not found"))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected code 404, got %d", resp.Code)
	}
}

func TestFailWithMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		FailWithMessage(c, http.StatusBadRequest, "custom error message")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Message != "custom error message" {
		t.Fatalf("expected message 'custom error message', got %s", resp.Message)
	}
}

func TestFailWithCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		FailWithCode(c, http.StatusConflict, 2001, "duplicate resource")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	var resp APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusConflict || resp.Code != 2001 || resp.Message != "duplicate resource" {
		t.Fatalf("unexpected response: status=%d response=%+v", w.Code, resp)
	}
}

func TestSuccessPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		SuccessPagination(c, []string{"a", "b"}, 1, 10, 25)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp ResponsePagination
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Pagination.TotalPages != 3 {
		t.Fatalf("expected 3 total pages, got %d", resp.Pagination.TotalPages)
	}
	if resp.Pagination.PageNo != 1 {
		t.Fatalf("expected pageNo 1, got %d", resp.Pagination.PageNo)
	}
	if resp.Pagination.TotalCount != 25 {
		t.Fatalf("expected totalCount 25, got %d", resp.Pagination.TotalCount)
	}
}

func TestSuccessPaginationWithCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		SuccessPaginationWithCode(c, 1002, "page ready", []string{"a"}, 2, 10, 21)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	var resp ResponsePagination
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != 1002 || resp.Message != "page ready" || resp.Pagination.TotalPages != 3 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
