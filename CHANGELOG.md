# Changelog

All notable changes to this project will be documented in this file.

## [0.0.3] - 2026-07-21

### Upgrade notes
- Existing exported functions and fields remain available, so most consumers remain source-compatible
- **JWT**: Tokens without `exp` are rejected by default. Set `AllowMissingExpiration: true` only as a temporary migration option
- **JWT**: Claims are no longer flattened into arbitrary Gin Context keys. Migrate `c.Get("claim")` calls to `jwt.ClaimsFromContext(c)` or `jwt.UsernameFromContext(c)`
- **ErrorHandler**: Private `err.Error()` text is no longer returned to clients. Mark safe errors as `gin.ErrorTypePublic` or configure `ErrorHandlerWithMapper`
- **CORS**: `cors.New([]string{"*"})` no longer enables credentials. Configure trusted origins explicitly for credentialed requests
- **Log**: No logger still means no output; a configured logger is enabled by default and can be toggled with `SetEnabled`

### Added
- **Log**: 新增 `SetEnabled` / `IsEnabled`。未配置 logger 时默认关闭；配置 logger 后默认开启，并支持并发安全地动态开关
- **JWT**: 新增 issuer、audience、leeway 与 legacy 无过期时间 token 的显式兼容配置
- **ErrorHandler**: 新增 public/private 错误隔离及自定义错误映射
- **Response**: 新增带独立业务码的 success、pagination 和 fail helpers
- **Tests**: 补充日志并发切换、JWT 安全边界、CORS 预检及错误信息隔离测试

### Changed
- **JWT**: 严格限制为配置的 HMAC 算法，默认要求 `exp`，生成 token 时不再修改调用方 claims
- **JWT**: Claims 使用包级命名空间保存，不再平铺并覆盖 Gin Context 中的其他值
- **JWT**: 认证失败响应不再暴露底层解析器错误
- **CORS**: 修复 wildcard origin 与 credentials 的无效组合，并校验预检 method/header
- **ErrorHandler**: 私有错误默认返回通用 500 消息，详细信息仅写入日志

### Fixed
- **Log**: 修复运行时配置或切换 logger 时的全局数据竞争
- **CORS**: 被拒绝的动态 Origin 响应现在也包含 `Vary: Origin`，避免共享缓存污染

## [0.0.1] - 2026-04-26

### Added
- **Log**: 新增 `log` 包 — 分级日志接口（Error/Warn/Info/Debug），支持标准库和自定义 Logger（slog、logrus 等），默认 discard 不输出
- **JWT**: 中间件自动记录认证失败日志（Missing header / Invalid format / Invalid token）
- **CORS**: 中间件自动记录来源被拦截日志
- **Examples**: 新增 `examples/main.go` 完整示例，演示 log、response、jwt、cors 所有功能及 slog 接入方式

### Changed
- **General**: 引入 `log` 包后，所有中间件在触发拦截时自动输出 Warn 级别日志，方便追踪调用方的 API 使用情况
- **Response**: Refactored `ResponsePagination` to nest pagination details into a dedicated `PaginationInfo` struct for a cleaner API structure.
- **Response**: Renamed `TotalSize` to `TotalCount` in pagination responses to follow common API conventions.
- **General**: Unified all middleware error responses to use the `response` package's standard format.
- **General**: Updated the entire codebase to use Go 1.18+ `any` keyword instead of `interface{}`.
- **JWT**: Introduced private `contextKey` type for Gin context keys to prevent collisions with other middleware or business logic.
- **JWT**: `ClaimsFromContext` and `UsernameFromContext` now prioritize retrieval using typed context keys while maintaining backward compatibility for string keys.
- **CORS**: Added `Vary: Origin` header to responses when `AllowedOrigins` is dynamic to prevent caching issues.
- **CORS**: Implemented O(1) lookup for `AllowedOrigins` using a map for better performance.

### Fixed
- **CORS**: Correctly handle requests with empty `Origin` headers by continuing the middleware chain.
