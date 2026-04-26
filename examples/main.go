package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/jackman0925/gin-middleware/cors"
	ginj "github.com/jackman0925/gin-middleware/jwt"
	ginlog "github.com/jackman0925/gin-middleware/log"
	"github.com/jackman0925/gin-middleware/response"
)

func main() {
	// ============================================================
	// 1. log — 启用日志，支持两种模式
	// ============================================================

	// 模式 A: 使用标准库日志（自带 [gin-middleware] 前缀）
	ginlog.SetStdLogger(ginlog.LevelDebug)
	ginlog.Infof("server starting at %s", time.Now().Format(time.RFC3339))

	// 模式 B: 接入第三方 logger（以 slog 为例）
	// 注释掉模式 A，取消注释此处即可切换
	/*
		slogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		ginlog.SetLogger(&slogAdapter{l: slogger}, ginlog.LevelDebug)
	*/

	// ============================================================
	// 2. response — 统一 API 响应格式
	// ============================================================
	//   APIResponse{ Code, Message, Data }
	//   ResponsePagination{ Code, Message, Data, Pagination{ PageNo, PageSize, TotalCount, TotalPages } }

	// ============================================================
	// 3. jwt — JWT 认证中间件
	// ============================================================
	secret := "demo-secret-key-at-least-32-chars-long!!"
	j := ginj.New(secret)

	// 验证配置
	if err := j.Validate(); err != nil {
		slog.Error("jwt config invalid", "error", err)
		os.Exit(1)
	}

	// ============================================================
	// 4. cors — 跨域中间件
	// ============================================================
	// cors.New([]string{"http://localhost:3000"}) — 指定来源
	// cors.AllowAll() — 全部允许（开发）

	gin.SetMode(gin.DebugMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// cors 中间件
	r.Use(cors.New([]string{"http://localhost:3000"}))

	// ---------- 公开路由 ----------

	// Health check
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{"status": "ok", "uptime": time.Now().Format(time.RFC3339)})
	})

	// Login — 演示 ginj.GenerateTokenWithUsername + response.Success
	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Fail(c, http.StatusBadRequest, err)
			return
		}
		// 模拟验证
		if req.Password != "demo123" {
			response.FailWithMessage(c, http.StatusUnauthorized, "invalid username or password")
			return
		}

		// 生成 token，携带额外 metadata
		token, err := j.GenerateTokenWithUsername(req.Username, map[string]any{
			"role": "admin",
			"id":   1001,
		})
		if err != nil {
			response.Fail(c, http.StatusInternalServerError, err)
			return
		}

		response.Success(c, gin.H{
			"token":    token,
			"username": req.Username,
			"expires":  time.Now().Add(j.Config.Expiration).Format(time.RFC3339),
		})
	})

	// Parse token（非中间件模式）— 演示 ginj.ParseToken
	r.POST("/parse", func(c *gin.Context) {
		var req struct {
			Token string `json:"token" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Fail(c, http.StatusBadRequest, err)
			return
		}
		claims, err := j.ParseToken(req.Token)
		if err != nil {
			response.FailWithMessage(c, http.StatusBadRequest, err.Error())
			return
		}
		response.Success(c, claims)
	})

	// ---------- 需要认证的路由 ----------

	auth := r.Group("/api/v1")
	auth.Use(j.Middleware())
	{
		// Profile — 演示 ginj.ClaimsFromContext / ginj.UsernameFromContext
		auth.GET("/profile", func(c *gin.Context) {
			username, _ := ginj.UsernameFromContext(c)
			claims, _ := ginj.ClaimsFromContext(c)
			response.Success(c, gin.H{
				"username": username,
				"claims":   claims,
			})
		})

		// Users — 演示 response.SuccessPagination
		auth.GET("/users", func(c *gin.Context) {
			users := []map[string]any{
				{"id": 1, "name": "Alice", "role": "admin"},
				{"id": 2, "name": "Bob", "role": "editor"},
				{"id": 3, "name": "Charlie", "role": "viewer"},
			}
			response.SuccessPagination(c, users, 1, 10, len(users))
		})

		// Error — 演示 response.FailWithMessage
		auth.GET("/error", func(c *gin.Context) {
			response.FailWithMessage(c, http.StatusTeapot, "I'm a teapot (demo error)")
		})

		// Logout — 演示 response.Success
		auth.POST("/logout", func(c *gin.Context) {
			username, _ := ginj.UsernameFromContext(c)
			ginlog.Infof("user %s logged out", username)
			response.Success(c, gin.H{"message": "logout successful"})
		})
	}

	// ---------- 自定义 JWT + CORS 配置 ----------
	// 演示 ginj.NewWithConfig / ginj.Config / cors.NewWithConfig / cors.Config
	customJWT := ginj.NewWithConfig(ginj.Config{
		Secret:          "another-secret-key-for-api-v2-at-least-32!!",
		TokenHeaderName: "X-API-Token",
		TokenPrefix:     "Token",
		Expiration:      time.Hour * 24,
		SigningMethod:   jwt.SigningMethodHS256, // 来自 golang-jwt
	})
	customCORS := cors.NewWithConfig(cors.Config{
		AllowedOrigins:   []string{"https://app.example.com"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Content-Type", "X-API-Token"},
		AllowCredentials: true,
		MaxAge:           3600,
	})

	v2 := r.Group("/api/v2")
	v2.Use(customCORS, customJWT.Middleware())
	{
		v2.GET("/status", func(c *gin.Context) {
			username, _ := ginj.UsernameFromContext(c)
			response.Success(c, gin.H{
				"version": "v2",
				"user":    username,
				"message": "custom JWT config (X-API-Token header)",
			})
		})
	}

	slog.Info("demo server started", "addr", "http://localhost:8080")
	slog.Info("try: curl -X POST http://localhost:8080/login -d '{\"username\":\"admin\",\"password\":\"demo123\"}'")
	r.Run(":8080")
}

// ---------- slog adapter ----------
// 演示如何实现 log.Logger 接口，将 gin-middleware 的日志接入 slog

type slogAdapter struct {
	l *slog.Logger
}

func (a *slogAdapter) Errorf(format string, v ...any) { a.l.ErrorContext(context.Background(), fmt.Sprintf(format, v...)) }
func (a *slogAdapter) Warnf(format string, v ...any)  { a.l.WarnContext(context.Background(), fmt.Sprintf(format, v...)) }
func (a *slogAdapter) Infof(format string, v ...any)  { a.l.InfoContext(context.Background(), fmt.Sprintf(format, v...)) }
func (a *slogAdapter) Debugf(format string, v ...any) { a.l.DebugContext(context.Background(), fmt.Sprintf(format, v...)) }
