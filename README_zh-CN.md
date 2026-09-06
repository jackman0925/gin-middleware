# gin-middleware

简体中文 | [English](README.md)

一组面向生产环境的 Gin Web 框架中间件。

## 功能

- **JWT 认证**：支持 Token 生成、解析以及 Gin 认证中间件，可配置密钥、有效期和签名算法。
- **CORS**：可配置的跨域资源共享中间件。
- **Request**：在 Gin 校验前填充默认值的 JSON 请求绑定助手。
- **Response**：统一的 API 响应格式助手。
- **Log**：支持日志级别和自定义后端的日志接口。
- **ErrorHandler**：统一捕获 Gin Context 错误，并安全地转换为 JSON 响应。

## 安装

```bash
go get github.com/jackman0925/gin-middleware
```

## 从 v0.0.2 及更早版本升级

新版本保留了已有的导出函数和字段，大多数项目升级后仍可正常编译。不过，为了提供更安全的生产默认值，部分运行时行为发生了有意调整。升级前请检查以下内容。

### JWT 有效期

JWT 校验现在默认要求 Token 包含 `exp` 声明。本库生成的 Token 本身已经包含 `exp`，可以继续正常使用。

如果其他服务或旧系统签发的是永不过期 Token，可以临时开启兼容选项：

```go
j := jwt.NewWithConfig(jwt.Config{
    Secret:                 "your-secret-key-must-have-32-bytes",
    AllowMissingExpiration: true,
})
```

`AllowMissingExpiration` 只建议在迁移期间使用。更推荐让 Token 签发方补充 `exp`，然后移除此配置。

### Gin Context 中的 JWT Claims

JWT Claims 不再复制到任意 Gin Context Key，以免覆盖其他中间件写入的数据。旧代码如果直接读取：

```go
adminID, exists := c.Get("adminID")
```

请改为通过统一入口读取：

```go
claims, exists := jwt.ClaimsFromContext(c)
if exists {
    adminID := claims["adminID"]
}
```

用户名可以直接使用 `jwt.UsernameFromContext(c)` 获取。

### 错误响应

`errorhandler.ErrorHandler()` 不再向客户端返回私有错误的 `err.Error()`，默认返回 `internal server error`，避免泄漏内部实现、数据库信息或文件路径。

只有确认内容可以安全展示给客户端时，才应将错误标记为公开错误：

```go
c.Error(errors.New("可以安全展示给客户端的信息")).SetType(gin.ErrorTypePublic)
```

需要自定义 HTTP 状态码和错误消息时，请使用 `errorhandler.ErrorHandlerWithMapper`。

### CORS 通配 Origin 与凭证

`cors.New([]string{"*"})` 不再默认开启凭证，因为浏览器不接受以下响应头组合：

```text
Access-Control-Allow-Origin: *
Access-Control-Allow-Credentials: true
```

如果业务需要携带 Cookie 或其他凭证，建议明确列出可信 Origin。自定义配置同时使用通配 Origin 和 credentials 时，中间件会回显当前请求的 Origin，并设置 `Vary: Origin`，但生产环境仍建议使用明确的 Origin 列表。

### 日志

未配置 logger 时，日志仍然默认关闭且不会产生输出。调用 `SetLogger` 或 `SetStdLogger` 后，日志默认开启；需要在运行期间关闭时，可以调用 `log.SetEnabled(false)`。

## 使用方法

### 日志

所有中间件包共用同一套日志接口。

默认不配置 logger，日志不会输出，无需进行任何初始化：

```go
import (
    "github.com/jackman0925/gin-middleware/cors"
    "github.com/jackman0925/gin-middleware/jwt"
)
```

使用标准库日志：

```go
import "github.com/jackman0925/gin-middleware/log"

log.SetStdLogger(log.LevelInfo)
```

支持以下日志级别：

- `LevelError`：仅错误日志。
- `LevelWarn`：警告和错误日志。
- `LevelInfo`：信息、警告和错误日志。
- `LevelDebug`：全部日志。

配置 logger 后默认开启，也可以随时关闭和恢复：

```go
log.SetEnabled(false)
log.SetEnabled(true)

enabled := log.IsEnabled()
```

调用 `log.SetLogger(nil, log.LevelError)` 会移除当前 logger 并关闭日志。logger 配置和动态开关均支持并发调用。

接入自定义 logger，以 glog 为例：

```go
type glogAdapter struct{}

func (glogAdapter) Errorf(f string, v ...any) { glog.Errorf(f, v...) }
func (glogAdapter) Warnf(f string, v ...any)  { glog.Warnf(f, v...) }
func (glogAdapter) Infof(f string, v ...any)  { glog.Infof(f, v...) }
func (glogAdapter) Debugf(f string, v ...any) { glog.Debugf(f, v...) }

log.SetLogger(glogAdapter{}, log.LevelDebug)
```

### JSON 请求绑定与默认值

当 JSON 请求字段需要在校验前设置默认值时，使用 `request` 包：

```go
import "github.com/jackman0925/gin-middleware/request"

type CreateUserRequest struct {
    Name    string `json:"name" binding:"required"`
    Page    int    `json:"page" default:"1" binding:"required,min=1"`
    Enabled bool   `json:"enabled" default:"true"`
}

func createUser(c *gin.Context) {
    var req CreateUserRequest
    if err := request.BindJSON(c, &req); err != nil {
        response.Fail(c, http.StatusBadRequest, err)
        return
    }
    // 未传 page 和 enabled 时，会在校验前分别填充为 1 和 true。
}
```

`BindJSON` 的执行顺序是：JSON 解码 → 按 `default:"..."` 标签填充默认值 → 调用 Gin validator。因此 `default` 可以与 `binding:"required"` 同时使用。客户端明确传入的 `false`、`0` 和空字符串会被保留，不会被默认值覆盖。

支持 string、bool、有符号/无符号整数、float、`time.Duration`（例如 `default:"5s"`）和 RFC3339 格式的 `time.Time`。slice、map、array 和 struct 的默认值需使用 JSON 字面量。

请求体会被缓存到 Gin 的 `BodyBytesKey`，后续仍可使用 `ShouldBindBodyWithJSON` 读取。对于非 JSON 对象可调用 `SetReqDefaults`；该函数会把零值视作未设置，因此 JSON 请求优先使用 `BindJSON`。

### JWT 认证

使用默认配置创建 JWT 中间件。默认签名算法为 HS256，有效期为 72 小时，Secret 至少需要 32 字节：

```go
import "github.com/jackman0925/gin-middleware/jwt"

j := jwt.New("your-32-char-or-longer-secret-key")
```

生成 Token：

```go
token, err := j.GenerateTokenWithUsername("admin", map[string]any{
    "adminID": 1,
    "role":    "admin",
})
if err != nil {
    // 处理错误
}
```

注册 Gin 中间件：

```go
r := gin.Default()

admin := r.Group("/admin")
admin.Use(j.Middleware())
{
    admin.GET("/dashboard", func(c *gin.Context) {
        username, ok := jwt.UsernameFromContext(c)
        if !ok {
            c.Status(http.StatusUnauthorized)
            return
        }
        c.JSON(http.StatusOK, gin.H{"username": username})
    })
}
```

自定义配置：

```go
import (
    "time"

    jwtlib "github.com/golang-jwt/jwt/v5"
    ginjwt "github.com/jackman0925/gin-middleware/jwt"
)

j := ginjwt.NewWithConfig(ginjwt.Config{
    Secret:          "your-secret-key-must-have-32-bytes",
    TokenHeaderName: "Authorization",
    TokenPrefix:     "Bearer",
    Expiration:      24 * time.Hour,
    SigningMethod:   jwtlib.SigningMethodHS256,
    Issuer:          "my-service",
    Audience:        "my-client",
    Leeway:          30 * time.Second,
})
```

JWT 解析只接受配置的 HMAC 算法，并默认要求 `exp`。Claims 保存在包专属的 Gin Context Key 下，请通过以下函数获取：

```go
claims, ok := jwt.ClaimsFromContext(c)
username, ok := jwt.UsernameFromContext(c)
```

### CORS

允许指定 Origin：

```go
import "github.com/jackman0925/gin-middleware/cors"

r := gin.Default()
r.Use(cors.New([]string{
    "https://example.com",
    "https://app.example.com",
}))
```

开发环境允许全部 Origin：

```go
r.Use(cors.AllowAll())
```

自定义配置：

```go
r.Use(cors.NewWithConfig(cors.Config{
    AllowedOrigins:   []string{"https://example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           86400,
}))
```

中间件会校验预检请求中的 Method 和 Header。动态 Origin 响应会设置 `Vary: Origin`，避免共享缓存错误复用跨域结果。

### 错误处理

将错误处理中间件放在业务 Handler 之前：

```go
import "github.com/jackman0925/gin-middleware/errorhandler"

r.Use(gin.Recovery())
r.Use(errorhandler.ErrorHandler())
```

业务 Handler 可以通过 `c.Error()` 提交错误：

```go
r.GET("/users/:id", func(c *gin.Context) {
    user, err := findUser(c.Param("id"))
    if err != nil {
        c.Error(err)
        return
    }
    response.Success(c, user)
})
```

私有错误的详细内容只会发送给已配置的 logger，客户端收到通用 500 消息。需要自定义映射时：

```go
r.Use(errorhandler.ErrorHandlerWithMapper(func(err *gin.Error) (int, string) {
    if errors.Is(err.Err, ErrNotFound) {
        return http.StatusNotFound, "资源不存在"
    }
    return http.StatusInternalServerError, "服务内部错误"
}))
```

### 统一响应

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

r.GET("/api/items/:id", func(c *gin.Context) {
    item, err := getItem(c.Param("id"))
    if err != nil {
        response.Fail(c, http.StatusNotFound, err)
        return
    }
    response.Success(c, item)
})
```

默认成功响应格式：

```json
{
  "code": 0,
  "message": "success",
  "data": {}
}
```

如果 API 使用独立业务码，可以使用：

```go
response.SuccessWithCode(c, 1001, "查询成功", data)
response.SuccessPaginationWithCode(c, 1002, "查询成功", data, page, pageSize, total)
response.FailWithCode(c, http.StatusConflict, 2001, "资源已存在")
```

HTTP 状态码和响应中的业务 `code` 相互独立。

## 测试

```bash
go test -race ./...
go vet ./...
```

项目 CI 会自动运行 race 单元测试和 `go vet`。

## 许可证

MIT
