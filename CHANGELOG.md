# Changelog

All notable changes to this project will be documented in this file.

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
