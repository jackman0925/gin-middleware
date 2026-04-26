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
	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessPagination returns a paginated 200 OK JSON response
func SuccessPagination(c *gin.Context, data any, pageNo, pageSize, totalCount int) {
	totalPages := calcTotalPages(totalCount, pageSize)
	c.JSON(http.StatusOK, ResponsePagination{
		Code:    0,
		Message: "success",
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
	c.AbortWithStatusJSON(status, APIResponse{
		Code:    status,
		Message: fmt.Sprintf("Request failed: %v", err),
	})
}

// FailWithMessage returns an error JSON response with a custom message and aborts
func FailWithMessage(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, APIResponse{
		Code:    status,
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

