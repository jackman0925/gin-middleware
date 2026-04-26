# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased] - 2026-04-26

### Added
- **JWT**: Introduced private `contextKey` type for Gin context keys to prevent collisions with other middleware or business logic.
- **CORS**: Added `Vary: Origin` header to responses when `AllowedOrigins` is dynamic to prevent caching issues.
- **CORS**: Implemented O(1) lookup for `AllowedOrigins` using a map for better performance.

### Changed
- **Response**: Refactored `ResponsePagination` to nest pagination details into a dedicated `PaginationInfo` struct for a cleaner API structure.
- **Response**: Renamed `TotalSize` to `TotalCount` in pagination responses to follow common API conventions.
- **General**: Unified all middleware error responses to use the `response` package's standard format.
- **General**: Updated the entire codebase to use Go 1.18+ `any` keyword instead of `interface{}`.
- **JWT**: `ClaimsFromContext` and `UsernameFromContext` now prioritize retrieval using typed context keys while maintaining backward compatibility for string keys.

### Fixed
- **CORS**: Correctly handle requests with empty `Origin` headers by continuing the middleware chain.
