# gin-middleware

A collection of production-ready middleware for the Gin web framework.

## Features

- **JWT Authentication** — Token generation, parsing, and Gin middleware with configurable secrets, expiration, and signing methods
- **CORS** — Configurable cross-origin resource sharing middleware
- **Response** — Standardized API response formatting helpers
- **Log** — Leveled logging interface with pluggable backends

## Installation

```bash
go get github.com/jackman0925/gin-middleware
```

## Usage

### Logging

All middleware packages share a common logging interface.

**By default, logging is disabled (discard)** — no output, zero overhead. If you don't need logs, just don't import the `log` package:

```go
// 什么都不需要做 — 日志默认关闭
import (
    "github.com/jackman0925/gin-middleware/jwt"
    "github.com/jackman0925/gin-middleware/cors"
)
```

If you want to see logs:

```go
import "github.com/jackman0925/gin-middleware/log"

// 标准库日志
log.SetStdLogger(log.LevelInfo)

// 接入 glog（github.com/jackman0925/glog）— 只需 4 行
type glogAdapter struct{}
func (glogAdapter) Errorf(f string, v ...any) { glog.Errorf(f, v...) }
func (glogAdapter) Warnf(f string, v ...any)  { glog.Warnf(f, v...) }
func (glogAdapter) Infof(f string, v ...any)  { glog.Infof(f, v...) }
func (glogAdapter) Debugf(f string, v ...any) { glog.Debugf(f, v...) }
log.SetLogger(glogAdapter{}, log.LevelDebug)
```

### JWT Authentication

```go
import "github.com/jackman0925/gin-middleware/jwt"

// Create JWT middleware with default config (HS256, 72h expiration)
j := jwt.New("your-32-char+ secret key here!!")

// Generate a token
token, err := j.GenerateTokenWithUsername("admin", map[string]interface{}{
    "adminID": 1,
    "role":    "admin",
})

// Use as Gin middleware
r := gin.Default()
admin := r.Group("/admin")
admin.Use(j.Middleware())
{
    admin.GET("/dashboard", func(c *gin.Context) {
        username, _ := jwt.UsernameFromContext(c)
        c.JSON(200, gin.H{"username": username})
    })
}
```

Custom configuration:

```go
j := jwt.NewWithConfig(jwt.Config{
    Secret:          "your-secret-key",
    TokenHeaderName: "Authorization",
    TokenPrefix:     "Bearer",
    Expiration:      time.Hour * 24,
    SigningMethod:   jwt.SigningMethodHS256,
})
```

### CORS

```go
import "github.com/jackman0925/gin-middleware/cors"

r := gin.Default()

// Allow specific origins
r.Use(cors.New([]string{"https://example.com", "https://app.example.com"}))

// Or allow all (development)
r.Use(cors.AllowAll())
```

Custom configuration:

```go
r.Use(cors.NewWithConfig(cors.Config{
    AllowedOrigins:   []string{"https://example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           86400,
}))
```

### Response Helpers

```go
import "github.com/jackman0925/gin-middleware/response"

r.GET("/api/users", func(c *gin.Context) {
    users := getUsers()
    response.Success(c, users)
})

r.GET("/api/products", func(c *gin.Context) {
    products := getProducts()
    response.SuccessPagination(c, products, page, pageSize, total)
})

r.GET("/api/item/:id", func(c *gin.Context) {
    item, err := getItem(id)
    if err != nil {
        response.Fail(c, http.StatusNotFound, err)
        return
    }
    response.Success(c, item)
})
```

## License

MIT
