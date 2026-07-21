// Package response provides standard API response helpers for Gin.
package response

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse defines the standard JSON response structure
type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// PaginationInfo defines the details of a paginated response
type PaginationInfo struct {
	PageNo     int `json:"pageNo"`
	PageSize   int `json:"pageSize"`
	TotalCount int `json:"totalCount"`
	TotalPages int `json:"totalPages"`
}

// ResponsePagination defines a paginated response structure
type ResponsePagination struct {
	Code       int            `json:"code"`
	Message    string         `json:"message"`
	Data       any            `json:"data,omitempty"`
	Pagination PaginationInfo `json:"pagination"`
}

// Success returns a 200 OK JSON response
func Success(c *gin.Context, data any) {
	SuccessWithCode(c, 0, "success", data)
}

// SuccessWithCode returns a 200 response with an application-specific code and
// message. It keeps the business code independent from the HTTP status.
func SuccessWithCode(c *gin.Context, code int, message string, data any) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    code,
		Message: message,
		Data:    data,
	})
}

// SuccessPagination returns a paginated 200 OK JSON response
func SuccessPagination(c *gin.Context, data any, pageNo, pageSize, totalCount int) {
	SuccessPaginationWithCode(c, 0, "success", data, pageNo, pageSize, totalCount)
}

// SuccessPaginationWithCode returns a paginated 200 response with an
// application-specific code and message.
func SuccessPaginationWithCode(c *gin.Context, code int, message string, data any, pageNo, pageSize, totalCount int) {
	totalPages := calcTotalPages(totalCount, pageSize)
	c.JSON(http.StatusOK, ResponsePagination{
		Code:    code,
		Message: message,
		Data:    data,
		Pagination: PaginationInfo{
			PageNo:     pageNo,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPages: totalPages,
		},
	})
}

// Fail returns an error JSON response and aborts the request
func Fail(c *gin.Context, status int, err error) {
	FailWithCode(c, status, status, fmt.Sprintf("Request failed: %v", err))
}

// FailWithMessage returns an error JSON response with a custom message and aborts
func FailWithMessage(c *gin.Context, status int, message string) {
	FailWithCode(c, status, status, message)
}

// FailWithCode returns an error response while keeping the application code
// independent from the HTTP status.
func FailWithCode(c *gin.Context, status, code int, message string) {
	c.AbortWithStatusJSON(status, APIResponse{
		Code:    code,
		Message: message,
	})
}

// calcTotalPages calculates total number of pages
func calcTotalPages(totalCount, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	if totalCount%pageSize == 0 {
		return totalCount / pageSize
	}
	return (totalCount / pageSize) + 1
}
