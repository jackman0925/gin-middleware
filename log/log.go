// Package log provides a leveled logging interface for gin-middleware.
//
// Users can set a custom logger via SetLogger(), or use the default
// stdlog logger that writes to the standard library log package.
// By default, logging is disabled (discard logger).
package log

import stdlog "log"

// Level represents the severity of a log message.
type Level int

const (
	// LevelError logs only errors.
	LevelError Level = iota
	// LevelWarn logs warnings and errors.
	LevelWarn
	// LevelInfo logs info, warnings, and errors.
	LevelInfo
	// LevelDebug logs all messages including debug.
	LevelDebug
)

// Logger defines the interface for logging. Implement this interface
// to plug in your own logger (slog, logrus, zap, etc.).
type Logger interface {
	Errorf(format string, v ...any)
	Warnf(format string, v ...any)
	Infof(format string, v ...any)
	Debugf(format string, v ...any)
}

type leveledLogger struct {
	level Level
}

func (l *leveledLogger) Errorf(format string, v ...any) {
	if l.level >= LevelError {
		stdlog.Printf("[gin-middleware] [ERROR] "+format, v...)
	}
}

func (l *leveledLogger) Warnf(format string, v ...any) {
	if l.level >= LevelWarn {
		stdlog.Printf("[gin-middleware] [WARN] "+format, v...)
	}
}

func (l *leveledLogger) Infof(format string, v ...any) {
	if l.level >= LevelInfo {
		stdlog.Printf("[gin-middleware] [INFO] "+format, v...)
	}
}

func (l *leveledLogger) Debugf(format string, v ...any) {
	if l.level >= LevelDebug {
		stdlog.Printf("[gin-middleware] [DEBUG] "+format, v...)
	}
}

type discardLogger struct{}

func (discardLogger) Errorf(format string, v ...any) {}
func (discardLogger) Warnf(format string, v ...any)  {}
func (discardLogger) Infof(format string, v ...any)  {}
func (discardLogger) Debugf(format string, v ...any) {}

// levelFilteredLogger wraps a custom Logger and filters by level
type levelFilteredLogger struct {
	Logger
	level Level
}

func (l *levelFilteredLogger) Errorf(format string, v ...any) {
	if l.level >= LevelError {
		l.Logger.Errorf(format, v...)
	}
}

func (l *levelFilteredLogger) Warnf(format string, v ...any) {
	if l.level >= LevelWarn {
		l.Logger.Warnf(format, v...)
	}
}

func (l *levelFilteredLogger) Infof(format string, v ...any) {
	if l.level >= LevelInfo {
		l.Logger.Infof(format, v...)
	}
}

func (l *levelFilteredLogger) Debugf(format string, v ...any) {
	if l.level >= LevelDebug {
		l.Logger.Debugf(format, v...)
	}
}

var global Logger = discardLogger{}

// SetLogger sets the global logger used by all middleware.
// Pass nil to disable logging (default).
func SetLogger(logger Logger, level Level) {
	if logger == nil {
		global = discardLogger{}
		return
	}
	// Wrap the custom logger with level filtering
	global = &levelFilteredLogger{Logger: logger, level: level}
}

// SetStdLogger sets a default stdlib-based logger at the given level.
// Pass LevelDebug for maximum verbosity, LevelError for minimum.
func SetStdLogger(level Level) {
	global = &leveledLogger{level: level}
}

// GetLogger returns the current global logger.
func GetLogger() Logger {
	return global
}

// Errorf logs an error message.
func Errorf(format string, v ...any) {
	global.Errorf(format, v...)
}

// Warnf logs a warning message.
func Warnf(format string, v ...any) {
	global.Warnf(format, v...)
}

// Infof logs an info message.
func Infof(format string, v ...any) {
	global.Infof(format, v...)
}

// Debugf logs a debug message.
func Debugf(format string, v ...any) {
	global.Debugf(format, v...)
}
